package workqueue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/symcn/api"
	"github.com/symcn/pkg/metrics"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type reconcileException struct {
	done    chan struct{}
	count   int
	requeue api.NeedRequeue
	after   time.Duration
	sleep   time.Duration
	err     error
}

func (r *reconcileException) Reconcile(item ktypes.NamespacedName) (api.NeedRequeue, time.Duration, error) {
	klog.Infof("mock Reconcile:%s", item.String())
	if r.sleep > 0 {
		time.Sleep(r.sleep)
	}
	if r.count < 1 {
		close(r.done)
		return api.Done, 0, nil
	}
	r.count--
	return r.requeue, r.after, r.err
}

func TestNewQueueException(t *testing.T) {
	t.Run("return error", func(t *testing.T) {
		done := make(chan struct{}, 0)

		qc := NewQueueConfig(&reconcileException{done: done, count: 2, err: errors.New("mock error")})
		qc.Name = "return_error"
		queue, err := Completed(qc).NewQueue()
		if err != nil {
			t.Error(err)
			return
		}

		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		go func() {
			queue.Start(ctx)
		}()

		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: "mock error"})
		<-done
	})

	t.Run("return after", func(t *testing.T) {
		done := make(chan struct{}, 0)
		qc := NewQueueConfig(&reconcileException{done: done, count: 5, after: time.Microsecond * 100})
		qc.Name = "return_after"
		queue, err := Completed(qc).NewQueue()
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		go func() {
			queue.Start(ctx)
		}()

		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: "mock error"})
		<-done
	})

	t.Run("return requeue", func(t *testing.T) {
		done := make(chan struct{}, 0)
		qc := NewQueueConfig(&reconcileException{done: done, count: 2, requeue: api.Requeue})
		qc.Name = "return_requeue"
		queue, err := Completed(qc).NewQueue()
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		go func() {
			queue.Start(ctx)
		}()

		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: "mock error"})
		<-done
	})

	t.Run("type unexpected", func(t *testing.T) {
		done := make(chan struct{}, 0)
		qc := NewQueueConfig(&reconcileException{done: done})
		qc.Name = "unexpected_type"
		queue, err := Completed(qc).NewQueue()
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()
		go func() {
			queue.Start(ctx)
		}()

		queue.Add("unexpected_type")
		time.Sleep(time.Millisecond * 200)
	})

	t.Run("add after shutdown", func(t *testing.T) {
		done := make(chan struct{}, 0)
		qc := NewQueueConfig(&reconcileException{done: done, sleep: time.Millisecond * 100})
		qc.Name = "add_after_shutdown"
		queue, err := Completed(qc).NewQueue()
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		go func() {
			queue.Start(ctx)
		}()
		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: "mock error"})
		cancel()
		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: "mock error"})
	})
}

func TestNewMetrics(t *testing.T) {
	server := startHTTPPrometheus(t)
	defer func() {
		server.Shutdown(context.Background())
	}()

	done := make(chan struct{}, 0)
	count := 100
	qc := NewQueueConfig(&reconcile{done: done, count: count, err: errors.New("mock error")})
	qc.Name = "benchmark"
	queue, err := Completed(qc).NewQueue()
	if err != nil {
		t.Error(err)
		return
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go func() {
		queue.Start(ctx)
	}()

	for i := 0; i < count; i++ {
		queue.Add(ktypes.NamespacedName{Namespace: "default", Name: fmt.Sprintf("item_%d", i)})
		// workqueue_return_requeue_ name_return_requeue_reconcile_fail_total
	}
	<-done
}

type reconcile struct {
	done  chan struct{}
	count int
	sleep time.Duration
	err   error
}

func (r *reconcile) Reconcile(item ktypes.NamespacedName) (api.NeedRequeue, time.Duration, error) {
	klog.Infof("mock Reconcile:%s", item.String())
	r.count--
	if r.count < 1 {
		if r.count == 0 {
			close(r.done)
		}
		return api.Done, 0, nil
	}
	switch r.count % 4 {
	case 0:
		return api.Requeue, 0, nil
	case 1:
		return api.Done, time.Millisecond * 20, nil
	case 2:
		return api.Done, 0, errors.New("mock error")
	case 3:
		time.Sleep(time.Millisecond * 10)
		return api.Done, 0, nil
	}
	return api.Done, 0, nil
}

// startHTTPPrometheus start http server with prometheus route
func startHTTPPrometheus(t *testing.T) *http.Server {
	server := &http.Server{
		Addr: ":8080",
	}
	mux := http.NewServeMux()
	metrics.RegisterHTTPHandler(func(pattern string, handler http.Handler) {
		mux.Handle(pattern, handler)
	})
	server.Handler = mux

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !strings.EqualFold(err.Error(), "http: Server closed") {
				t.Error(err)
			}
		}
		t.Log("http shutdown")
	}()
	return server
}
