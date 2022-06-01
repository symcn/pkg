package workqueue

import (
	"context"
	"errors"
	"time"

	"github.com/symcn/api"
	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
)

// ratelimit queue set
var (
	defaultQueueName             = "symcn-queue"
	defaultGotInterval           = time.Second * 1
	defaultRateLimitTimeInterval = time.Second * 1
	defaultRateLimitTimeMax      = time.Second * 60
	defaultRateLimit             = 10
	defaultRateBurst             = 100
	defaultThreadiness           = 1
)

type ReconcilerType int

const (
	Normal ReconcilerType = iota
	Wrapper
	Event
)

type QueueConfig struct {
	ctx                   context.Context
	Name                  string
	GotInterval           time.Duration
	RateLimitTimeInterval time.Duration
	RateLimitTimeMax      time.Duration
	RateLimit             int
	RateBurst             int
	Threadiness           int

	RT      ReconcilerType
	Do      api.Reconciler
	WrapDo  api.WrapReconciler
	EventDo api.EventReonciler
}

type completedConfig struct {
	*QueueConfig
}

// CompletedConfig wrapper workqueue
type CompletedConfig struct {
	*completedConfig
}

type queue struct {
	*CompletedConfig
	Workqueue workqueue.RateLimitingInterface
	Stats     *stats
}

type queueObj struct {
	*CompletedConfig
	Workqueue workqueue.RateLimitingInterface
	Stats     *stats
}

// NewQueueConfig build standard queue
func NewQueueConfig(reconcile api.Reconciler) *QueueConfig {
	qc := &QueueConfig{
		Name:                  defaultQueueName,
		GotInterval:           defaultGotInterval,
		RateLimitTimeInterval: defaultRateLimitTimeInterval,
		RateLimitTimeMax:      defaultRateLimitTimeMax,
		RateLimit:             defaultRateLimit,
		RateBurst:             defaultRateBurst,
		Threadiness:           defaultThreadiness,
		RT:                    Normal,
		Do:                    reconcile,
	}

	return qc
}

// NewWrapQueueConfig build queue which request with clustername
func NewWrapQueueConfig(name string, reconcile api.WrapReconciler) *QueueConfig {
	qc := &QueueConfig{
		Name:                  name,
		GotInterval:           defaultGotInterval,
		RateLimitTimeInterval: defaultRateLimitTimeInterval,
		RateLimitTimeMax:      defaultRateLimitTimeMax,
		RateLimit:             defaultRateLimit,
		RateBurst:             defaultRateBurst,
		Threadiness:           defaultThreadiness,
		RT:                    Wrapper,
		WrapDo:                reconcile,
	}

	return qc
}

// NewEventQueueConfig build queue which request with clustername and event function
func NewEventQueueConfig(name string, reconcile api.EventReonciler) *QueueConfig {
	qc := &QueueConfig{
		Name:                  name,
		GotInterval:           defaultGotInterval,
		RateLimitTimeInterval: defaultRateLimitTimeInterval,
		RateLimitTimeMax:      defaultRateLimitTimeMax,
		RateLimit:             defaultRateLimit,
		RateBurst:             defaultRateBurst,
		Threadiness:           defaultThreadiness,
		RT:                    Event,
		EventDo:               reconcile,
	}

	return qc
}

func Completed(qc *QueueConfig) *CompletedConfig {
	cc := &CompletedConfig{&completedConfig{qc}}

	if cc.Name == "" {
		cc.Name = defaultQueueName
	}

	if cc.GotInterval < defaultGotInterval {
		cc.GotInterval = defaultGotInterval
	}

	if cc.Threadiness < 1 {
		cc.Threadiness = defaultThreadiness
	}

	return cc
}

// NewQueue build queue
func (cc *CompletedConfig) NewQueue() (api.WorkQueue, error) {
	switch cc.RT {
	case Normal:
		if cc.Do == nil {
			return nil, errors.New("NewQueueConfig should use standard Reconciler")
		}
	case Wrapper:
		if cc.WrapDo == nil {
			return nil, errors.New("newWrapQueueConfig should use WrapReconciler")
		}
	case Event:
		if cc.EventDo == nil {
			return nil, errors.New("NewEventQueueConfig should use WrapReconciler")
		}
	default:
		return nil, errors.New("not support ReconcilerType")
	}

	stats, err := buildStats(cc.Name)
	if err != nil {
		return nil, err
	}

	q := &queue{
		CompletedConfig: cc,
		Stats:           stats,
		Workqueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewMaxOfRateLimiter(
				workqueue.NewItemExponentialFailureRateLimiter(cc.RateLimitTimeInterval, cc.RateLimitTimeMax),
				&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(float64(cc.RateLimit)), cc.RateBurst)},
			),
			cc.Name,
		),
	}

	return q, nil
}
