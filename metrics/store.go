package metrics

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/api"
	"k8s.io/klog/v2"
)

// MaxLabelCount max label count limit
const MaxLabelCount = 20

var (
	defaultEndpoint          = "/metrics"
	defaultSummaryObjectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.001, 0.99: 0.001}
	defaultSummaryMaxAge     = time.Minute * 1
	defaultStore             *store
	// ErrLabelCountExceeded error label count exceeded
	ErrLabelCountExceeded = fmt.Errorf("label count exceeded, max is %d", MaxLabelCount)
)

type store struct {
	l       sync.RWMutex
	metrics map[string]api.Metrics
}

type metrics struct {
	typ    string
	prefix string
	col    []prometheus.Collector
}

func init() {
	defaultStore = &store{
		metrics: make(map[string]api.Metrics, 100),
	}
}

func (m *metrics) Counter(key string) prometheus.Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: flattenKey(m.prefix + key),
	})
	m.registerPrometheus(counter)
	return counter
}

func (m *metrics) Gauge(key string) prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: flattenKey(m.prefix + key),
	})
	m.registerPrometheus(gauge)
	return gauge
}

func (m *metrics) Histogram(key string, buckets []float64) prometheus.Histogram {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    flattenKey(m.prefix + key),
		Buckets: buckets,
	})
	m.registerPrometheus(histogram)
	return histogram
}

func (m *metrics) Summary(key string, objectives map[float64]float64) prometheus.Summary {
	if len(objectives) == 0 {
		objectives = defaultSummaryObjectives
	}

	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       flattenKey(m.prefix + key),
		Objectives: objectives,
		MaxAge:     defaultSummaryMaxAge,
	})
	m.registerPrometheus(summary)
	return summary
}

func (m *metrics) UnregisterAll() {
	for _, col := range m.col {
		prometheus.Unregister(col)
	}
}

func (m *metrics) registerPrometheus(c prometheus.Collector) {
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("registry prometheus failed: %v", r)
			debug.PrintStack()
		}
	}()
	prometheus.MustRegister(c)
	m.col = append(m.col, c)
}

// Only [a-zA-Z0-9:_] are valid in metric names, any other characters should be sanitized to an underscore.
var flattenRegexp = regexp.MustCompile("[^a-zA-Z0-9_:]")

func flattenKey(key string) string {
	return flattenRegexp.ReplaceAllString(key, "_")
}

func resetAll() {
	defaultStore.l.Lock()
	defer defaultStore.l.Unlock()

	for _, m := range defaultStore.metrics {
		m.UnregisterAll()
	}
	defaultStore.metrics = make(map[string]api.Metrics, 100)
}

func sortedLabels(labels map[string]string) (keys, values []string) {
	keys = make([]string, 0, len(labels))
	values = make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values = append(values, labels[k])
	}
	return
}

func fullName(typ string, labels map[string]string) (fullName string) {
	if len(labels) == 0 {
		return typ
	}

	keys, values := sortedLabels(labels)
	pair := make([]string, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		pair = append(pair, keys[i]+"."+values[i])
	}
	fullName = typ + "." + strings.Join(pair, ".")
	return
}
