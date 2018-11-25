# prometheus-query-proxy

This works as a reverse proxy in front of multiple Prometheus.

## Installation

```
go get github.com/ryotarai/prometheus-query-proxy
```

## Configuration

```yaml
datasources:
- url: "http://localhost:9090"
  resolution: '15s' # a.k.a. scrape interval
  retention: '360h' # optional
- url: "http://localhost:9091"
  resolution: '1m' # a.k.a. scrape interval
  retention: '1440h' # optional
```

## Endpoints

### `/api/v1/query_range`

A datasource is picked up based on:

```
(Time of now) - (Datasource retention) <= ("start" query param)
  AND (Datasource resolution) <= ("step" query param)
```

If there are multiple datasources, this selects a datasource whose resolution is the shortest.

### `/api/v1/query`

A datasource is picked up based on:

```
(Time of now) - (Datasource retention) <= ("time" query param)
```

If there are multiple datasources, this selects a datasource whose resolution is the shortest.

### `/api/v1/label/KEY/values`

This requests to all datasources and merge them into one response.
