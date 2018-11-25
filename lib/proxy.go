package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var digitsRe = regexp.MustCompile("^[0-9\\.]+$")
var labelValuesPathRe = regexp.MustCompile("^/api/v1/label/[^/]+/values$")

type Proxy struct {
	config *Config
}

func NewProxy(c *Config) *Proxy {
	p := &Proxy{
		config: c,
	}
	return p
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/api/v1/query_range" {
		p.handleQueryRange(rw, req)
	} else if req.URL.Path == "/api/v1/query" {
		p.handleQuery(rw, req)
	} else if labelValuesPathRe.MatchString(req.URL.Path) {
		p.handleLabelNameValues(rw, req)
	} else {
		http.NotFound(rw, req)
	}
}

func (p *Proxy) handleQueryRange(rw http.ResponseWriter, req *http.Request) {
	step, err := parsePromDuration(req.URL.Query().Get("step"))
	if err != nil {
		rw.WriteHeader(400)
		fmt.Fprintf(rw, "%s\n", err)
		return
	}

	start, err := parsePromTime(req.URL.Query().Get("start"))
	if err != nil {
		rw.WriteHeader(400)
		fmt.Fprintf(rw, "%s\n", err)
		return
	}

	end, err := parsePromTime(req.URL.Query().Get("end"))
	if err != nil {
		rw.WriteHeader(400)
		fmt.Fprintf(rw, "%s\n", err)
		return
	}

	var targetDs *DatasourceConfig
	for _, ds := range p.config.Datasources {
		var zero time.Duration

		if ds.Retention != zero {
			retentionStart := time.Now().Add(ds.Retention * -1)
			if start.Before(retentionStart) || end.Before(retentionStart) {
				continue
			}
		}

		stepResolutionGap := step.Nanoseconds() - ds.Resolution.Nanoseconds()
		if stepResolutionGap < 0 {
			continue
		}

		if targetDs == nil || stepResolutionGap < step.Nanoseconds()-targetDs.Resolution.Nanoseconds() {
			targetDs = ds
		}
	}

	if targetDs == nil {
		rw.WriteHeader(400)
		fmt.Fprint(rw, "Datasource for the query is not found\n")
		return
	}

	log.Printf("request: query_range, datasource: %s, step: %s, start: %s, end: %s", targetDs.URL.String(), step, start, end)
	httputil.NewSingleHostReverseProxy(targetDs.URL).ServeHTTP(rw, req)
}

func (p *Proxy) handleQuery(rw http.ResponseWriter, req *http.Request) {
	t, err := parsePromTime(req.URL.Query().Get("time"))
	if err != nil {
		rw.WriteHeader(400)
		fmt.Fprintf(rw, "%s\n", err)
		return
	}

	var targetDs *DatasourceConfig
	for _, ds := range p.config.Datasources {
		var zero time.Duration

		if ds.Retention != zero {
			retentionStart := time.Now().Add(ds.Retention * -1)
			if t.Before(retentionStart) {
				continue
			}
		}
		if targetDs == nil || targetDs.Resolution > ds.Resolution {
			targetDs = ds
		}
	}

	if targetDs == nil {
		rw.WriteHeader(400)
		fmt.Fprint(rw, "Datasource for the query is not found\n")
		return
	}

	log.Printf("request: query, datasource: %s, time: %s", targetDs.URL.String(), t)
	httputil.NewSingleHostReverseProxy(targetDs.URL).ServeHTTP(rw, req)
}

type labelValuesResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func (p *Proxy) handleLabelNameValues(rw http.ResponseWriter, req *http.Request) {
	var wg sync.WaitGroup
	var mutex sync.Mutex

	labelsMap := map[string]struct{}{}
	for _, ds := range p.config.Datasources {
		wg.Add(1)
		go func(ds *DatasourceConfig) {
			u := *ds.URL
			u.Path = singleJoiningSlash(u.Path, req.URL.Path)
			r, err := http.NewRequest("GET", u.String(), http.NoBody)
			c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			r = r.WithContext(c)
			resp, err := http.DefaultClient.Do(r)
			if err != nil {
				log.Printf("Error getting %s: %s", u.String(), err)
				return
			}
			if resp.StatusCode != 200 {
				log.Printf("Error getting %s: status code %d", u.String(), resp.StatusCode)
			}

			lvr := labelValuesResponse{}
			d := json.NewDecoder(resp.Body)
			d.Decode(&lvr)
			resp.Body.Close()
			if lvr.Status != "success" {
				log.Printf("Ignoring response from %s because status is %s", u.String(), lvr.Status)
			}

			for _, l := range lvr.Data {
				mutex.Lock()
				labelsMap[l] = struct{}{}
				mutex.Unlock()
			}
			defer wg.Done()
		}(ds)
	}
	wg.Wait()

	labels := []string{}
	for l := range labelsMap {
		labels = append(labels, l)
	}

	sort.Slice(labels, func(i, j int) bool {
		return labels[i] < labels[j]
	})

	lvr := labelValuesResponse{
		Status: "success",
		Data:   labels,
	}
	e := json.NewEncoder(rw)
	err := e.Encode(lvr)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(rw, "Error serializing to JSON: %s\n", err)
		return
	}

	log.Printf("request: %s", req.URL.String())
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func isDigit(s string) bool {
	return digitsRe.MatchString(s)
}

func parsePromTime(s string) (time.Time, error) {
	var zero time.Time
	if isDigit(s) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, err
		}
		return time.Unix(0, int64(f*1000*1000*1000)), nil
	} else {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return zero, err
		}
		return t, nil
	}
}

func parsePromDuration(s string) (time.Duration, error) {
	var zero time.Duration
	if isDigit(s) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, err
		}
		return time.Duration(int64(f * 1000 * 1000 * 1000)), nil
	} else {
		d, err := time.ParseDuration(s)
		if err != nil {
			return zero, err
		}
		return d, nil
	}
}
