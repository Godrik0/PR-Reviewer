package metrics

type Metrics interface {
	IncHTTPRequests(method, path string, statusCode int)
	ObserveHTTPDuration(method, path string, duration float64)
}
