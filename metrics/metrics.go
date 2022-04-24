package metrics

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/symcn/api"
)

// NewMetrics returns a metrics
func NewMetrics(prefix string, constLabels map[string]string) (api.Metrics, error) {
	if len(constLabels) > MaxLabelCount {
		return nil, ErrLabelCountExceeded
	}

	defaultStore.l.Lock()
	defer defaultStore.l.Unlock()

	if col, ok := defaultStore.metrics[prefix]; ok {
		return col, nil
	}
	stats := &metrics{
		prefix:      strings.TrimRight(prefix, "_") + "_",
		constLabels: constLabels,
		col:         []prometheus.Collector{},
		metricVec:   map[string]*prometheus.MetricVec{},
	}
	defaultStore.metrics[prefix] = stats
	return stats, nil
}

// RegisterHTTPHandler register metrics with http mode
func RegisterHTTPHandler(f func(pattern string, handler http.Handler)) {
	f(defaultEndpoint, promhttp.Handler())
}
