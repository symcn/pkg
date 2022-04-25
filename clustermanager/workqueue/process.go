package workqueue

import (
	"fmt"
	"time"

	"github.com/symcn/api"
	ktypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type processFunc func(q *queue, obj interface{}) error

var processFactory = map[ReconcilerType]processFunc{
	Normal:  processReconcile,
	Wrapper: processWrapReconcile,
	Event:   processEventReconcile,
}

func processReconcile(q *queue, obj interface{}) error {
	req, ok := obj.(ktypes.NamespacedName)
	if !ok {
		q.Workqueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected types.NamespacedName in workqueue but got %#v", obj))
		q.Stats.UnExpectedObj.Inc()
		return nil
	}
	requeue, after, err := q.Do.Reconcile(q.ctx, req)
	return q.resultProcessing(requeue, after, err, obj)
}

func processWrapReconcile(q *queue, obj interface{}) error {
	req, ok := obj.(ktypes.NamespacedName)
	if !ok {
		q.Workqueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected types.NamespacedName in workqueue but got %#v", obj))
		q.Stats.UnExpectedObj.Inc()
		return nil
	}
	requeue, after, err := q.WrapDo.Reconcile(q.ctx, api.WrapNamespacedName{NamespacedName: req, QName: q.Name})
	return q.resultProcessing(requeue, after, err, obj)
}

func processEventReconcile(q *queue, obj interface{}) error {
	req, ok := obj.(api.EventRequest)
	if !ok {
		q.Workqueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected api.EventRequest in workqueue but got %#v", obj))
		q.Stats.UnExpectedObj.Inc()
		return nil
	}
	var (
		requeue api.NeedRequeue
		after   time.Duration
		err     error
	)
	switch req.EventType {
	case api.AddEvent:
		requeue, after, err = q.EventDo.OnAdd(q.ctx, q.Name, req.NewResource)
	case api.UpdateEvent:
		requeue, after, err = q.EventDo.OnUpdate(q.ctx, q.Name, req.OldResource, req.NewResource)
	case api.DeleteEvent:
		requeue, after, err = q.EventDo.OnDelete(q.ctx, q.Name, req.NewResource)
	default:
		q.Workqueue.Forget(obj)
		utilruntime.HandleError(fmt.Errorf("expected api.EventRequest Type but got %d", req.EventType))
		return nil
	}
	return q.resultProcessing(requeue, after, err, obj)
}
