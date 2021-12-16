package clustermanager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/configuration"
	"github.com/symcn/pkg/clustermanager/predicate"
	"github.com/symcn/pkg/clustermanager/workqueue"
	corev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	mockOpt = DefaultOptions()
)

func TestNewMultiClient(t *testing.T) {
	cli, err := NewMingleClient(configuration.BuildDefaultClusterCfgInfo("meta"), mockOpt)
	if err != nil {
		t.Error(err)
		return
	}
	clusterCfgManager := configuration.NewClusterCfgManagerWithCM(cli.GetKubeInterface(), "sym-admin", map[string]string{"ClusterOwner": "sym-admin"}, "kubeconfig.yaml", "status")

	mcc := NewMultiClientConfig()
	mcc.ClusterCfgManager = clusterCfgManager
	mcc.Options = mockOpt
	cc, err := Complete(mcc)
	if err != nil {
		t.Error(err)
		return
	}

	multiCli, err := cc.New()
	if err != nil {
		t.Error(err)
		return
	}
	err = multiCli.TriggerSync(&corev1.ConfigMap{})
	if err != nil {
		t.Error(err)
		return
	}

	qc := workqueue.NewQueueConfig(&reconcile{})
	qc.Name = "mockreconcile"
	queue, err := workqueue.Complted(qc).NewQueue()
	if err != nil {
		t.Error(err)
		return
	}
	/*
	 *     eventHandler := &mockEventHandler{}
	 *     err = multiCli.Watch(&corev1.Pod{}, queue, eventHandler, predicate.NamespacePredicate("*"))
	 *     if err != nil {
	 *         t.Error(err)
	 *         return
	 *     }
	 *
	 *     err = cli.Watch(&corev1.ConfigMap{}, queue, eventHandler, predicate.NamespacePredicate("*"))
	 *     if err != nil {
	 *         t.Error(err)
	 *         return
	 *     }
	 */
	multiCli.RegistryBeforAfterHandler(func(ctx context.Context, cli api.MingleClient) error {
		eventHandler := &mockEventHandler{}
		err := cli.Watch(&corev1.Pod{}, queue, eventHandler, predicate.NamespacePredicate("*"))
		if err != nil {
			t.Error(err)
			return err
		}

		err = cli.Watch(&corev1.ConfigMap{}, queue, eventHandler, predicate.NamespacePredicate("*"))
		if err != nil {
			t.Error(err)
			return err
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ch := make(chan struct{}, 0)
	go func() {
		err = queue.Start(ctx)
		if err != nil {
			t.Error(err)
		}
	}()

	go func() {
		err = multiCli.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		close(ch)
	}()

	syncCh := make(chan struct{}, 0)
	go func() {
		for !multiCli.HasSynced() {
			t.Log("wait sync")
			time.Sleep(time.Millisecond * 100)
		}
		close(syncCh)
	}()

	select {
	case <-ch:
	case <-syncCh:
	}
}

func TestMultiClientQueueLifeCycleWithClient(t *testing.T) {
	cli, err := NewMingleClient(configuration.BuildDefaultClusterCfgInfo("meta"), mockOpt)
	if err != nil {
		t.Error(err)
		return
	}
	clusterCfgManager := configuration.NewClusterCfgManagerWithCM(cli.GetKubeInterface(), "sym-admin", map[string]string{"ClusterOwner": "sym-admin"}, "kubeconfig.yaml", "status")

	mcc := NewMultiClientConfig()
	mcc.ClusterCfgManager = clusterCfgManager
	mcc.Options = mockOpt
	cc, err := Complete(mcc)
	if err != nil {
		t.Error(err)
		return
	}

	multiCli, err := cc.New()
	if err != nil {
		t.Error(err)
		return
	}

	sameLifeCycle := make(chan struct{}, 0)

	multiCli.RegistryBeforAfterHandler(func(ctx context.Context, cli api.MingleClient) error {
		queue, err := workqueue.Complted(workqueue.NewWrapQueueConfig(cli.GetClusterCfgInfo().GetName(), &wrapreconcile{})).NewQueue()
		if err != nil {
			return err
		}
		go queue.Start(ctx)

		go func() {
			<-ctx.Done()
			close(sameLifeCycle)
		}()

		eventHandler := &mockEventHandler{}
		err = cli.Watch(&corev1.Pod{}, queue, eventHandler, predicate.NamespacePredicate("*"))
		if err != nil {
			t.Error(err)
			return err
		}

		err = cli.Watch(&corev1.ConfigMap{}, queue, eventHandler, predicate.NamespacePredicate("*"))
		if err != nil {
			t.Error(err)
			return err
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.TODO())

	ch := make(chan struct{}, 0)
	go func() {
		err = multiCli.Start(ctx)
		if err != nil {
			t.Error(err)
		}
		close(ch)
	}()

	syncCh := make(chan struct{}, 0)
	go func() {
		for !multiCli.HasSynced() {
			t.Log("wait sync")
			time.Sleep(time.Millisecond * 100)
		}
		close(syncCh)
	}()

	select {
	case <-ch:
	case <-syncCh:
	}

	if len(multiCli.GetAll()) == 0 {
		close(sameLifeCycle)
	}

	cancel()

	select {
	case <-sameLifeCycle:
	case <-time.After(time.Second * 1):
		t.Error("beforStartHandleList context is not life cycle with client")
	}
}

type reconcile struct {
}

func (r *reconcile) Reconcile(req ktypes.NamespacedName) (requeue api.NeedRequeue, after time.Duration, err error) {
	fmt.Println(req.String())
	return api.Done, 0, nil
}

type wrapreconcile struct {
}

func (wr *wrapreconcile) Reconcile(req api.WrapNamespacedName) (requeue api.NeedRequeue, after time.Duration, err error) {
	fmt.Println(req.NamespacedName.String())
	return api.Done, 0, nil
}

type mockEventHandler struct {
}

func (t *mockEventHandler) Create(obj rtclient.Object, queue api.WorkQueue) {
	// gvks, b, err := mockOpt.Scheme.ObjectKinds(obj)
	// if err != nil {
	//     fmt.Println(err)
	//     return
	// }
	// if b {
	//     return
	// }
	// if len(gvks) == 1 {
	//     fmt.Println(gvks[0].Kind)
	// }
	queue.Add(ktypes.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()})
}

func (t *mockEventHandler) Update(oldObj, newObj rtclient.Object, queue api.WorkQueue) {
}

func (t *mockEventHandler) Delete(obj rtclient.Object, queue api.WorkQueue) {
}

func (t *mockEventHandler) Generic(obj rtclient.Object, queue api.WorkQueue) {
}
