package client

import (
	"context"

	"github.com/symcn/api"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	rtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	rtmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

type FakeClient struct {
	rtclient.WithWatch

	StopCh chan struct{}

	*Options
	ClusterCfg api.ClusterCfgInfo

	AddResourceEventHandlerFunc func(obj rtclient.Object, handler cache.ResourceEventHandler) error
	CreateFunc                  func(obj rtclient.Object, opts ...rtclient.CreateOption) error
	DeleteFunc                  func(obj rtclient.Object, opts ...rtclient.DeleteOption) error
	DeleteAllOfFunc             func(obj rtclient.Object, opts ...rtclient.DeleteAllOfOption) error
	GetFunc                     func(key ktypes.NamespacedName, obj rtclient.Object) error
	GetInformerFunc             func(obj rtclient.Object) (rtcache.Informer, error)
	HasSyncedFunc               func() bool
	ListFunc                    func(obj rtclient.ObjectList, opts ...rtclient.ListOption) error
	PatchFunc                   func(obj rtclient.Object, patch rtclient.Patch, opts ...rtclient.PatchOption) error
	SetIndexFieldFunc           func(obj rtclient.Object, field string, extractValue rtclient.IndexerFunc) error
	StatusUpdateFunc            func(obj rtclient.Object, opts ...rtclient.SubResourceUpdateOption) error
	UpdateFunc                  func(obj rtclient.Object, opts ...rtclient.UpdateOption) error
	AnnotatedEventfFunc         func(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{})
	EventFunc                   func(object runtime.Object, eventtype string, reason string, message string)
	EventfFunc                  func(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{})
	GetDynamicInterfaceFunc     func() dynamic.Interface
	GetKubeInterfaceFunc        func() kubernetes.Interface
	GetKubeRestConfigFunc       func() *rest.Config
	GetCtrlRtCacheFunc          func() rtcache.Cache
	GetCtrlRtClientFunc         func() rtclient.Client
	GetCtrlRtManagerFunc        func() rtmanager.Manager
	WatchFunc                   func(src rtclient.Object, queue api.WorkQueue, handler api.EventHandler, predicates ...api.Predicate) error
	GetClusterCfgInfoFunc       func() api.ClusterCfgInfo
	IsConnectedFunc             func() bool
}

func NewFackeClient(clusterCfg api.ClusterCfgInfo, opt *Options) (api.MingleClient, error) {
	return &FakeClient{
		WithWatch:  fake.NewFakeClient(),
		StopCh:     make(chan struct{}),
		Options:    opt,
		ClusterCfg: clusterCfg,
	}, nil
}

// AddResourceEventHandler implements api.MingleClient
func (f *FakeClient) AddResourceEventHandler(obj rtclient.Object, handler cache.ResourceEventHandler) error {
	if f.AddResourceEventHandlerFunc == nil {
		return nil
	}

	return f.AddResourceEventHandlerFunc(obj, handler)
}

// Create implements api.MingleClient
func (f *FakeClient) Create(obj rtclient.Object, opts ...rtclient.CreateOption) error {
	if f.CreateFunc == nil {
		return f.WithWatch.Create(context.TODO(), obj, opts...)
	}
	return f.CreateFunc(obj, opts...)
}

// Delete implements api.MingleClient
func (f *FakeClient) Delete(obj rtclient.Object, opts ...rtclient.DeleteOption) error {
	if f.DeleteFunc == nil {
		return f.WithWatch.Delete(context.TODO(), obj, opts...)
	}
	return f.DeleteFunc(obj, opts...)
}

// DeleteAllOf implements api.MingleClient
func (f *FakeClient) DeleteAllOf(obj rtclient.Object, opts ...rtclient.DeleteAllOfOption) error {
	if f.DeleteAllOfFunc == nil {
		return f.WithWatch.DeleteAllOf(context.TODO(), obj, opts...)
	}
	return f.DeleteAllOfFunc(obj, opts...)
}

// Get implements api.MingleClient
func (f *FakeClient) Get(key ktypes.NamespacedName, obj rtclient.Object) error {
	if f.GetFunc == nil {
		return f.WithWatch.Get(context.TODO(), key, obj)
	}
	return f.GetFunc(key, obj)
}

// GetInformer implements api.MingleClient
func (f *FakeClient) GetInformer(obj rtclient.Object) (rtcache.Informer, error) {
	if f.GetInformerFunc == nil {
		return nil, nil
	}
	return f.GetInformerFunc(obj)
}

// HasSynced implements api.MingleClient
func (f *FakeClient) HasSynced() bool {
	if f.HasSyncedFunc == nil {
		return true
	}
	return f.HasSyncedFunc()
}

// List implements api.MingleClient
func (f *FakeClient) List(obj rtclient.ObjectList, opts ...rtclient.ListOption) error {
	if f.ListFunc == nil {
		return f.WithWatch.List(context.TODO(), obj, opts...)
	}
	return f.ListFunc(obj, opts...)
}

// Patch implements api.MingleClient
func (f *FakeClient) Patch(obj rtclient.Object, patch rtclient.Patch, opts ...rtclient.PatchOption) error {
	if f.PatchFunc == nil {
		return f.WithWatch.Patch(context.TODO(), obj, patch, opts...)
	}
	return f.PatchFunc(obj, patch, opts...)
}

// SetIndexField implements api.MingleClient
func (f *FakeClient) SetIndexField(obj rtclient.Object, field string, extractValue rtclient.IndexerFunc) error {
	if f.SetIndexFieldFunc == nil {
		return nil
	}
	return f.SetIndexFieldFunc(obj, field, extractValue)
}

// StatusUpdate implements api.MingleClient
func (f *FakeClient) StatusUpdate(obj rtclient.Object, opts ...rtclient.SubResourceUpdateOption) error {
	if f.StatusUpdateFunc == nil {
		return f.WithWatch.Status().Update(context.TODO(), obj, opts...)
	}
	return f.StatusUpdateFunc(obj, opts...)
}

// Update implements api.MingleClient
func (f *FakeClient) Update(obj rtclient.Object, opts ...rtclient.UpdateOption) error {
	if f.UpdateFunc == nil {
		return f.WithWatch.Update(context.TODO(), obj, opts...)
	}
	return f.UpdateFunc(obj, opts...)
}

// AnnotatedEventf implements api.MingleClient
func (f *FakeClient) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{}) {
	if f.AnnotatedEventfFunc == nil {
		return
	}
	f.AnnotatedEventfFunc(object, annotations, eventtype, reason, messageFmt, args)
}

// Event implements api.MingleClient
func (f *FakeClient) Event(object runtime.Object, eventtype string, reason string, message string) {
	if f.EventFunc == nil {
		return
	}
	f.EventFunc(object, eventtype, reason, message)
}

// Eventf implements api.MingleClient
func (f *FakeClient) Eventf(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{}) {
	if f.EventfFunc == nil {
		return
	}
	f.EventfFunc(object, eventtype, reason, messageFmt, args...)
}

// GetDynamicInterface implements api.MingleClient
func (f *FakeClient) GetDynamicInterface() dynamic.Interface {
	if f.GetDynamicInterfaceFunc == nil {
		return nil
	}
	return f.GetDynamicInterfaceFunc()
}

// GetKubeInterface implements api.MingleClient
func (f *FakeClient) GetKubeInterface() kubernetes.Interface {
	if f.GetKubeInterfaceFunc == nil {
		return nil
	}
	return f.GetKubeInterfaceFunc()
}

// GetKubeRestConfig implements api.MingleClient
func (f *FakeClient) GetKubeRestConfig() *rest.Config {
	if f.GetKubeRestConfigFunc == nil {
		return nil
	}
	return f.GetKubeRestConfigFunc()
}

// GetCtrlRtCache implements api.MingleClient
func (f *FakeClient) GetCtrlRtCache() rtcache.Cache {
	if f.GetCtrlRtCacheFunc == nil {
		return nil
	}
	return f.GetCtrlRtCacheFunc()
}

// GetCtrlRtClient implements api.MingleClient
func (f *FakeClient) GetCtrlRtClient() rtclient.Client {
	if f.GetCtrlRtClientFunc == nil {
		return nil
	}
	return f.GetCtrlRtClientFunc()
}

// GetCtrlRtManager implements api.MingleClient
func (f *FakeClient) GetCtrlRtManager() rtmanager.Manager {
	if f.GetCtrlRtManagerFunc == nil {
		return nil
	}
	return f.GetCtrlRtManagerFunc()
}

// Watch implements api.MingleClient
func (f *FakeClient) Watch(src rtclient.Object, queue api.WorkQueue, handler api.EventHandler, predicates ...api.Predicate) error {
	if f.WatchFunc == nil {
		return nil
	}
	return f.WatchFunc(src, queue, handler, predicates...)
}

// GetClusterCfgInfo implements api.MingleClient
func (f *FakeClient) GetClusterCfgInfo() api.ClusterCfgInfo {
	if f.GetClusterCfgInfoFunc == nil {
		return f.ClusterCfg
	}
	return f.GetClusterCfgInfoFunc()
}

// IsConnected implements api.MingleClient
func (f *FakeClient) IsConnected() bool {
	if f.IsConnectedFunc == nil {
		return true
	}
	return f.IsConnectedFunc()
}

// Start implements api.MingleClient
func (f *FakeClient) Start(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case <-f.StopCh:
		return nil
	}
}

// Stop implements api.MingleClient
func (f *FakeClient) Stop() {
	close(f.StopCh)
}
