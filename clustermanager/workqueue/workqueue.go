package workqueue

import (
	"context"
	"fmt"
	"time"

	"github.com/symcn/api"
	"golang.org/x/time/rate"
	ktypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// ratelimit queue set
var (
	RateLimitTimeInterval = time.Second * 1
	RateLimitTimeMax      = time.Second * 60
	RateLimit             = 10
	RateBurst             = 100
)

// Queue wrapper workqueue
type Queue struct {
	name        string
	threadiness int
	gotIntervel time.Duration
	workqueue   workqueue.RateLimitingInterface
	stats       *stats
	Do          api.Reconciler
}

// NewQueue build queue
func NewQueue(reconcile api.Reconciler, name string, threadiness int, gotInterval time.Duration) (api.WorkQueue, error) {
	stats, err := buildStats(name)
	if err != nil {
		return nil, err
	}

	return &Queue{
		name:        name,
		threadiness: threadiness,
		gotIntervel: gotInterval,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(RateLimitTimeInterval, RateLimitTimeMax),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(RateLimit), RateBurst)},
		), name),
		stats: stats,
		Do:    reconcile,
	}, nil
}

// Add add obj to queue
func (q *Queue) Add(item interface{}) {
	q.workqueue.Add(item)
}

// Start will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (q *Queue) Start(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer q.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Infof("Starting %s WrapQueue workers", q.name)
	// Launch two workers to process Foo resources
	for i := 0; i < q.threadiness; i++ {
		go wait.UntilWithContext(ctx, q.runWorker, q.gotIntervel)
	}

	klog.Infof("Started %s WrapQueue workers", q.name)
	<-ctx.Done()
	klog.Infof("Shutting down %s WrapQueue workers", q.name)
	return nil
}

func (q *Queue) runWorker(ctx context.Context) {
	for q.processNextWorkItem() {
	}
}

func (q *Queue) processNextWorkItem() bool {
	obj, shutdown := q.workqueue.Get()
	if shutdown {
		return false
	}
	q.stats.Dequeue.Inc()

	start := time.Now()
	defer func() {
		q.stats.ReconcileDuration.Observe(float64(time.Since(start)))
	}()

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer q.workqueue.Done(obj)

		// TODO: invoke Reconcile
		var req ktypes.NamespacedName
		var ok bool
		if req, ok = obj.(ktypes.NamespacedName); !ok {
			q.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected types.NamespacedName in workqueue but got %#v", obj))
			q.stats.UnExpectedObj.Inc()
			return nil
		}

		// invoke Reconcile
		requeue, after, err := q.Do.Reconcile(req)
		if err != nil {
			q.workqueue.AddRateLimited(req)
			q.stats.ReconcileFail.Inc()
			q.stats.RequeueRateLimit.Inc()
			return nil
		}

		q.stats.ReconcileSucc.Inc()

		if after > 0 {
			q.workqueue.Forget(obj)
			q.workqueue.AddAfter(req, after)
			q.stats.RequeueAfter.Inc()
			return nil
		}
		if requeue == api.Requeue {
			q.workqueue.AddRateLimited(req)
			q.stats.RequeueRateLimit.Inc()
			return nil
		}

		q.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}
