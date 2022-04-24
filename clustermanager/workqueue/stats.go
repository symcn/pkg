package workqueue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/symcn/pkg/metrics"
)

var (
	metricTypePre      = "workqueue_"
	workqueueLabelName = "queuename"
)

// metrics key with labels
const (
	DequeueTotal          = "dequeue_total"
	UnExpectedObjTotal    = "unexpected_obj_total"
	ReconcileSuccTotal    = "reconcile_succ_total"
	ReconcileFailTotal    = "reconcile_fail_total"
	ReconcileTimeDuration = "reconcile_duration"
	RequeueAfterTotal     = "requeue_after_total"
	RequeueRateLimitTotal = "requeue_rate_limit_total"
)

type stats struct {
	Dequeue           prometheus.Counter
	UnExpectedObj     prometheus.Counter
	ReconcileSucc     prometheus.Counter
	ReconcileFail     prometheus.Counter
	ReconcileDuration prometheus.Histogram
	RequeueAfter      prometheus.Counter
	RequeueRateLimit  prometheus.Counter
}

func buildStats(queueName string) (*stats, error) {
	metric, err := metrics.NewMetrics(metricTypePre, nil)
	if err != nil {
		return nil, err
	}
	dynamicLabels := map[string]string{
		workqueueLabelName: queueName,
	}

	return &stats{
		Dequeue:           metric.CounterWithLabels(DequeueTotal, dynamicLabels),
		UnExpectedObj:     metric.CounterWithLabels(UnExpectedObjTotal, dynamicLabels),
		ReconcileSucc:     metric.CounterWithLabels(ReconcileSuccTotal, dynamicLabels),
		ReconcileFail:     metric.CounterWithLabels(ReconcileFailTotal, dynamicLabels),
		ReconcileDuration: metric.SummaryWithLables(ReconcileTimeDuration, nil, dynamicLabels),
		RequeueAfter:      metric.CounterWithLabels(RequeueAfterTotal, dynamicLabels),
		RequeueRateLimit:  metric.CounterWithLabels(RequeueRateLimitTotal, dynamicLabels),
	}, nil
}
