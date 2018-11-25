package lib

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
)

// StartCLI launches CLI command
func StartCLI() {
	options := parseCLIOptions()
	err := validateCLIOptions(options)
	if err != nil {
		log.Fatal(err)
	}

	config, err := LoadConfig(options.configPath)
	if err != nil {
		log.Printf("Error loading config: %s", err)
	}

	for _, ds := range config.Datasources {
		log.Printf("Datasource: %+v", *ds)
	}

	proxy := NewProxy(config)
	log.Printf("Listening %s", options.listen)
	err = http.ListenAndServe(options.listen, proxy)
	if err != nil {
		log.Fatal(err)
	}
}

type cliOptions struct {
	configPath string
	listen     string
}

func parseCLIOptions() cliOptions {
	listen := os.Getenv("PROM_QUERY_PROXY_LISTEN")
	if listen == "" {
		listen = ":8080"
	}

	options := cliOptions{}
	flag.StringVar(&options.configPath, "config", os.Getenv("PROM_QUERY_PROXY_CONFIG"), "Path to config file")
	flag.StringVar(&options.listen, "listen", listen, "Address to listen on")
	flag.Parse()
	return options
}

func validateCLIOptions(options cliOptions) error {
	if options.configPath == "" {
		return errors.New("-config option is required")
	}
	return nil
}
