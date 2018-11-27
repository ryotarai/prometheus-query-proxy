package lib

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Datasources []*DatasourceConfig `yaml:"datasources"`
}

type DatasourceConfig struct {
	URLStr        string `yaml:"url"`
	ResolutionStr string `yaml:"resolution"`
	RetentionStr  string `yaml:"retention"`
	StartTimeStr  string `yaml:"startTime"`

	URL        *url.URL      `yaml:"-"`
	Resolution time.Duration `yaml:"-"`
	Retention  time.Duration `yaml:"-"`
	StartTime  time.Time     `yaml:"-"`
}

func LoadConfig(path string) (*Config, error) {
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := Config{}
	yaml.UnmarshalStrict(in, &config)

	err = config.validate()
	if err != nil {
		return nil, err
	}

	err = config.parse()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) validate() error {
	for _, ds := range c.Datasources {
		if ds.ResolutionStr == "" {
			return fmt.Errorf("datasources[].resolution is required")
		}
		if ds.URLStr == "" {
			return fmt.Errorf("datasources[].url is required")
		}
	}

	return nil
}

func (c *Config) parse() error {
	for _, ds := range c.Datasources {
		if ds.ResolutionStr != "" {
			d, err := time.ParseDuration(ds.ResolutionStr)
			if err != nil {
				return err
			}
			ds.Resolution = d
		}

		if ds.RetentionStr != "" {
			d, err := time.ParseDuration(ds.RetentionStr)
			if err != nil {
				return err
			}
			ds.Retention = d
		}

		if ds.URLStr != "" {
			u, err := url.Parse(ds.URLStr)
			if err != nil {
				return err
			}
			ds.URL = u
		}

		if ds.StartTimeStr != "" {
			t, err := time.Parse(time.RFC3339, ds.StartTimeStr)
			if err != nil {
				return err
			}
			ds.StartTime = t
		}
	}

	return nil
}
