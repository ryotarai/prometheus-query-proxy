package lib

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProxyQueryPicksLowestResolution(t *testing.T) {
	ds1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds1")
	}))
	defer ds1.Close()
	ds2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds2")
	}))
	defer ds2.Close()
	ds3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds3")
	}))
	defer ds3.Close()

	url1, _ := url.Parse(ds1.URL)
	url2, _ := url.Parse(ds2.URL)
	url3, _ := url.Parse(ds3.URL)

	config := &Config{
		Datasources: []*DatasourceConfig{
			{
				URL:        url1,
				Resolution: time.Second,
				Retention:  time.Hour,
			},
			{
				URL:        url2,
				Resolution: time.Minute,
			},
			{
				URL:        url3,
				Resolution: time.Hour,
			},
		},
	}
	proxy := NewProxy(config)
	proxyServer := httptest.NewServer(proxy)

	u, _ := url.Parse(fmt.Sprintf("%s/api/v1/query", proxyServer.URL))
	q := u.Query()
	q.Add("time", time.Now().Add(-2*time.Hour).Format(time.RFC3339))
	u.RawQuery = q.Encode()

	resp, err := proxyServer.Client().Get(u.String())
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ds2", string(b))
}

func TestProxyQueryRangeBasedOnStep(t *testing.T) {
	ds1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds1")
	}))
	defer ds1.Close()
	ds2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds2")
	}))
	defer ds2.Close()
	ds3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds3")
	}))
	defer ds3.Close()

	url1, _ := url.Parse(ds1.URL)
	url2, _ := url.Parse(ds2.URL)
	url3, _ := url.Parse(ds3.URL)

	config := &Config{
		Datasources: []*DatasourceConfig{
			{
				URL:        url1,
				Resolution: time.Second,
			},
			{
				URL:        url2,
				Resolution: time.Minute,
			},
			{
				URL:        url3,
				Resolution: time.Minute * 10,
			},
		},
	}
	proxy := NewProxy(config)
	proxyServer := httptest.NewServer(proxy)

	u, _ := url.Parse(fmt.Sprintf("%s/api/v1/query_range", proxyServer.URL))
	q := u.Query()
	q.Add("start", time.Now().Add(time.Hour*-1).Format(time.RFC3339))
	q.Add("end", time.Now().Format(time.RFC3339))
	q.Add("step", "2m")
	u.RawQuery = q.Encode()
	t.Log(u.String())

	resp, err := proxyServer.Client().Get(u.String())
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ds2", string(b))
}

func TestProxyQueryRangeBasedOnTime(t *testing.T) {
	ds1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds1")
	}))
	defer ds1.Close()
	ds2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ds2")
	}))
	defer ds2.Close()

	url1, _ := url.Parse(ds1.URL)
	url2, _ := url.Parse(ds2.URL)

	config := &Config{
		Datasources: []*DatasourceConfig{
			{
				URL:        url1,
				Resolution: time.Minute,
				Retention:  time.Hour,
			},
			{
				URL:        url2,
				Resolution: time.Minute,
			},
		},
	}
	proxy := NewProxy(config)
	proxyServer := httptest.NewServer(proxy)

	u, _ := url.Parse(fmt.Sprintf("%s/api/v1/query_range", proxyServer.URL))
	q := u.Query()
	q.Add("start", time.Now().Add(time.Hour*-2).Format(time.RFC3339))
	q.Add("end", time.Now().Format(time.RFC3339))
	q.Add("step", "2m")
	u.RawQuery = q.Encode()
	t.Log(u.String())

	resp, err := proxyServer.Client().Get(u.String())
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ds2", string(b))
}

func TestProxyLabelNameValues(t *testing.T) {
	ds1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "success", "data": ["foo", "ds1/%s"]}`, r.URL.String())
	}))
	defer ds1.Close()
	ds2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status": "success", "data": ["foo", "ds2/%s"]}`, r.URL.String())
	}))
	defer ds2.Close()

	url1, _ := url.Parse(ds1.URL)
	url2, _ := url.Parse(ds2.URL)

	config := &Config{
		Datasources: []*DatasourceConfig{
			{
				URL:        url1,
				Resolution: time.Minute,
			},
			{
				URL:        url2,
				Resolution: time.Minute,
			},
		},
	}
	proxy := NewProxy(config)
	proxyServer := httptest.NewServer(proxy)

	u, _ := url.Parse(fmt.Sprintf("%s/api/v1/label/foo/values", proxyServer.URL))
	resp, err := proxyServer.Client().Get(u.String())
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"status":"success","data":["ds1//api/v1/label/foo/values","ds2//api/v1/label/foo/values","foo"]}`+"\n", string(b))
}
