package metrics

import (
	"errors"
	"fmt"
	"regexp"
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
	l sync.Mutex

	typ         string
	prefix      string
	constLabels prometheus.Labels
	col         []prometheus.Collector
	metricVec   map[string]*prometheus.MetricVec
}

func init() {
	defaultStore = &store{
		metrics: make(map[string]api.Metrics, 100),
	}
}

func (m *metrics) Counter(name string) prometheus.Counter {
	counter, err := m.CounterWithLabelsWithError(name, nil)
	if err != nil {
		klog.Error(err)
	}
	return counter
}

func (m *metrics) CounterWithLabels(name string, dynamicLabels map[string]string) prometheus.Counter {
	counter, err := m.CounterWithLabelsWithError(name, dynamicLabels)
	if err != nil {
		klog.Error(err)
	}
	return counter
}

func (m *metrics) CounterWithLabelsWithError(name string, dynamicLabels map[string]string) (prometheus.Counter, error) {
	m.l.Lock()
	mv, ok := m.metricVec[name]
	if !ok {
		keys, _ := sortedLabels(dynamicLabels)

		counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        flattenKey(m.prefix + name),
			ConstLabels: m.constLabels,
		}, keys)

		if len(m.metricVec) == 0 {
			m.metricVec = map[string]*prometheus.MetricVec{}
		}
		mv = counterVec.MetricVec
		m.metricVec[name] = mv
		m.registerPrometheus(counterVec)
	}
	m.l.Unlock()

	metrics, err := mv.GetMetricWith(dynamicLabels)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		return nil, errors.New("GetMetricsWithLabels return nil")
	}
	counter, ok := metrics.(prometheus.Counter)
	if !ok {
		return nil, errors.New("repeat registry Counter with labels")
	}
	return counter, nil
}

func (m *metrics) Gauge(name string) prometheus.Gauge {
	gauge, err := m.GaugeWithLabelsWithError(name, nil)
	if err != nil {
		klog.Error(err)
	}
	return gauge
}

func (m *metrics) GaugeWithLabels(name string, dynamicLabels map[string]string) prometheus.Gauge {
	gauge, err := m.GaugeWithLabelsWithError(name, dynamicLabels)
	if err != nil {
		klog.Error(err)
	}
	return gauge
}

func (m *metrics) GaugeWithLabelsWithError(name string, dynamicLabels map[string]string) (prometheus.Gauge, error) {
	m.l.Lock()
	mv, ok := m.metricVec[name]
	if !ok {
		keys, _ := sortedLabels(dynamicLabels)

		gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:        flattenKey(m.prefix + name),
			ConstLabels: m.constLabels,
		}, keys)

		mv = gaugeVec.MetricVec
		if len(m.metricVec) == 0 {
			m.metricVec = map[string]*prometheus.MetricVec{}
		}
		m.metricVec[name] = mv
		m.registerPrometheus(gaugeVec)
	}
	m.l.Unlock()

	metrics, err := mv.GetMetricWith(dynamicLabels)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		return nil, errors.New("GetMetricsWithLabels return nil")
	}
	gauge, ok := metrics.(prometheus.Gauge)
	if !ok {
		return nil, errors.New("repeat registry Gauge with labels")
	}
	return gauge, nil
}

func (m *metrics) Histogram(name string, buckets []float64) prometheus.Histogram {
	histogram, err := m.HistogramWithLabelsWithError(name, buckets, nil)
	if err != nil {
		klog.Error(err)
	}
	return histogram
}

func (m *metrics) HistogramWithLabels(name string, buckets []float64, dynamicLabels map[string]string) prometheus.Histogram {
	histogram, err := m.HistogramWithLabelsWithError(name, buckets, dynamicLabels)
	if err != nil {
		klog.Error(err)
	}
	return histogram
}
func (m *metrics) HistogramWithLabelsWithError(name string, buckets []float64, dynamicLabels map[string]string) (prometheus.Histogram, error) {
	m.l.Lock()
	mv, ok := m.metricVec[name]
	if !ok {
		keys, _ := sortedLabels(dynamicLabels)
		histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        flattenKey(m.prefix + name),
			ConstLabels: m.constLabels,
			Buckets:     buckets,
		}, keys)

		if len(m.metricVec) == 0 {
			m.metricVec = map[string]*prometheus.MetricVec{}
		}
		mv = histogramVec.MetricVec
		m.metricVec[name] = mv
		m.registerPrometheus(histogramVec)
	}
	m.l.Unlock()

	metrics, err := mv.GetMetricWith(dynamicLabels)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		return nil, errors.New("GetMetricsWithLabels return nil")
	}
	histogram, ok := metrics.(prometheus.Histogram)
	if !ok {
		return nil, errors.New("repeat registry Histogram with labels")
	}
	return histogram, nil
}

func (m *metrics) Summary(name string, objectives map[float64]float64) prometheus.Summary {
	summary, err := m.SummaryWithLabelsWithError(name, objectives, nil)
	if err != nil {
		klog.Error(err)
	}
	return summary
}

func (m *metrics) SummaryWithLables(name string, objectives map[float64]float64, dynamicLabels map[string]string) prometheus.Summary {
	summary, err := m.SummaryWithLabelsWithError(name, objectives, dynamicLabels)
	if err != nil {
		klog.Error(err)
	}
	return summary
}

func (m *metrics) SummaryWithLabelsWithError(name string, objectives map[float64]float64, dynamicLabels map[string]string) (prometheus.Summary, error) {
	m.l.Lock()
	mv, ok := m.metricVec[name]
	if !ok {
		keys, _ := sortedLabels(dynamicLabels)

		if len(objectives) == 0 {
			objectives = defaultSummaryObjectives
		}
		summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:        flattenKey(m.prefix + name),
			ConstLabels: m.constLabels,
			Objectives:  objectives,
			MaxAge:      defaultSummaryMaxAge,
		}, keys)

		if len(m.metricVec) == 0 {
			m.metricVec = map[string]*prometheus.MetricVec{}
		}
		mv = summaryVec.MetricVec
		m.metricVec[name] = mv
		m.registerPrometheus(summaryVec)
	}
	m.l.Unlock()

	metrics, err := mv.GetMetricWith(dynamicLabels)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		return nil, errors.New("GetMetricsWithLabels return nil")
	}
	summary, ok := metrics.(prometheus.Summary)
	if !ok {
		return nil, errors.New("repeat registry Summary with labels")
	}
	return summary, nil
}

func (m *metrics) UnregisterAll() {
	for _, col := range m.col {
		prometheus.Unregister(col)
	}
	m.metricVec = nil
}

func (m *metrics) DeleteWithLabels(name string, labels map[string]string) bool {
	m.l.Lock()
	defer m.l.Unlock()

	mv, ok := m.metricVec[name]
	if !ok {
		return false
	}
	return mv.Delete(labels)
}

func (m *metrics) registerPrometheus(c prometheus.Collector) error {
	if err := prometheus.Register(c); err != nil {
		klog.Error(err)
		return err
	}
	m.col = append(m.col, c)
	return nil
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
	if len(labels) < 1 {
		return nil, nil
	}

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
