package workqueue

import (
	"context"
	"time"

	"github.com/symcn/api"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// Add add obj to queue
func (q *queue) Add(item interface{}) {
	q.Workqueue.Add(item)
}

// Start will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (q *queue) Start(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	defer q.Workqueue.ShutDown()

	q.ctx = ctx

	// Start the informer factories to begin populating the informer caches
	klog.Infof("Starting %s WrapQueue workers", q.Name)
	// Launch two workers to process Foo resources
	for i := 0; i < q.Threadiness; i++ {
		go wait.UntilWithContext(ctx, q.runWorker, q.GotInterval)
	}

	klog.Infof("Started %s WrapQueue workers", q.Name)
	<-ctx.Done()
	klog.Infof("Shutting down %s WrapQueue workers", q.Name)
	return nil
}

func (q *queue) runWorker(ctx context.Context) {
	for q.processNextWorkItem() {
	}
}

func (q *queue) processNextWorkItem() bool {
	obj, shutdown := q.Workqueue.Get()
	if shutdown {
		return false
	}
	q.Stats.Dequeue.Inc()

	start := time.Now()
	defer func() {
		klog.V(4).Info("===== queue->%s reconcile %s duration: %s", q.Name, time.Since(start))
		q.Stats.ReconcileDuration.Observe(float64(time.Since(start)))
	}()

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer q.Workqueue.Done(obj)

		if f, ok := processFactory[q.RT]; ok {
			return f(q, obj)
		}
		q.Workqueue.Forget(obj)
		klog.Error("Unsupport reconciler type.")
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (q *queue) resultProcessing(requeue api.NeedRequeue, after time.Duration, err error, obj interface{}) error {
	if err != nil {
		klog.Errorf("[workqueue] reconcile %+v (qname:%s) failed: %+v", obj, q.Name, err)
		// TODO: return error need add queue again?
		q.Workqueue.AddRateLimited(obj)
		q.Stats.ReconcileFail.Inc()
		q.Stats.RequeueRateLimit.Inc()
		return nil
	}

	q.Stats.ReconcileSucc.Inc()

	if after > 0 {
		q.Workqueue.Forget(obj)
		q.Workqueue.AddAfter(obj, after)
		q.Stats.RequeueAfter.Inc()
		return nil
	}
	if requeue == api.Requeue {
		q.Workqueue.AddRateLimited(obj)
		q.Stats.RequeueRateLimit.Inc()
		return nil
	}

	q.Workqueue.Forget(obj)
	return nil
}
