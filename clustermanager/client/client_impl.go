package client

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/handler"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	rtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	rtmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

// GetInformer fetches or constructs an informer for the given object that corresponds to a single
// API kind and resource.
func (c *client) GetInformer(obj rtclient.Object) (rtcache.Informer, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	informer, err := c.ctrlRtCache.GetInformer(ctx, obj)
	if err != nil {
		return nil, err
	}
	c.informerList = append(c.informerList, informer)
	return informer, nil
}

// AddResourceEventHandler
//  1. GetInformer
//  2. Adds an event handler to the shared informer using the shared informer's resync
//     period.  Events to a single handler are delivered sequentially, but there is no coordination
//     between different handlers.
func (c *client) AddResourceEventHandler(obj rtclient.Object, handler cache.ResourceEventHandler) error {
	informer, err := c.GetInformer(obj)
	if err != nil {
		return err
	}
	informer.AddEventHandler(handler)
	return nil
}

// IndexFields adds an index with the given field name on the given object type
// by using the given function to extract the value for that field.  If you want
// compatibility with the Kubernetes API server, only return one key, and only use
// fields that the API server supports.  Otherwise, you can return multiple keys,
// and "equality" in the field selector means that at least one key matches the value.
// The FieldIndexer will automatically take care of indexing over namespace
// and supporting efficient all-namespace queries.
func (c *client) SetIndexField(obj rtclient.Object, field string, extractValue rtclient.IndexerFunc) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtManager.GetFieldIndexer().IndexField(ctx, obj, field, extractValue)
}

// Watch takes events provided by a Source and uses the EventHandler to
// enqueue reconcile.Requests in response to the events.
//
// Watch may be provided one or more Predicates to filter events before
// they are given to the EventHandler.  Events will be passed to the
// EventHandler if all provided Predicates evaluate to true.
func (c *client) Watch(obj rtclient.Object, queue api.WorkQueue, evtHandler api.EventHandler, predicates ...api.Predicate) error {
	if queue == nil {
		return errors.New("api.WorkQueue is nil")
	}
	return c.AddResourceEventHandler(obj, handler.NewResourceEventHandler(queue, evtHandler, predicates...))
}

// HasSynced return true if all informers underlying store has synced
// !import if informerlist is empty, will return true
func (c *client) HasSynced() bool {
	if atomic.LoadInt32(&c.started) != 1 {
		// if not start, the informer will not synced
		return false
	}

	for _, informer := range c.informerList {
		if !informer.HasSynced() {
			return false
		}
	}
	return true
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster with timeout.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c *client) Get(key ktypes.NamespacedName, obj rtclient.Object) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Get(ctx, key, obj)
}

// Create saves the object obj in the Kubernetes cluster with timeout.
func (c *client) Create(obj rtclient.Object, opts ...rtclient.CreateOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Create(ctx, obj, opts...)
}

// Delete deletes the given obj from Kubernetes cluster with timeout.
func (c *client) Delete(obj rtclient.Object, opts ...rtclient.DeleteOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Delete(ctx, obj, opts...)
}

// Update updates the given obj in the Kubernetes cluster with timeout. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *client) Update(obj rtclient.Object, opts ...rtclient.UpdateOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Update(ctx, obj, opts...)
}

// Update updates the fields corresponding to the status subresource for the
// given obj with timeout. obj must be a struct pointer so that obj can be updated
// with the content returned by the Server.
func (c *client) StatusUpdate(obj rtclient.Object, opts ...rtclient.SubResourceUpdateOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Status().Update(ctx, obj, opts...)
}

// Patch patches the given obj in the Kubernetes cluster with timeout. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *client) Patch(obj rtclient.Object, patch rtclient.Patch, opts ...rtclient.PatchOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.Patch(ctx, obj, patch, opts...)
}

// DeleteAllOf deletes all objects of the given type matching the given options with timeout.
func (c *client) DeleteAllOf(obj rtclient.Object, opts ...rtclient.DeleteAllOfOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.DeleteAllOf(ctx, obj, opts...)
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c *client) List(obj rtclient.ObjectList, opts ...rtclient.ListOption) error {
	ctx, cancel := context.WithTimeout(context.TODO(), c.ExecTimeout)
	defer cancel()

	return c.ctrlRtClient.List(ctx, obj, opts...)
}

// Event constructs an event from the given information and puts it in the queue for sending.
// 'object' is the object this event is about. Event will make a reference-- or you may also
// pass a reference to the object directly.
// 'type' of this event, and can be one of Normal, Warning. New types could be added in future
// 'reason' is the reason this event is generated. 'reason' should be short and unique; it
// should be in UpperCamelCase format (starting with a capital letter). "reason" will be used
// to automate handling of events, so imagine people writing switch statements to handle them.
// You want to make that easy.
// 'message' is intended to be human readable.
//
// The resulting event will be created in the same namespace as the reference object.
func (c *client) Event(object runtime.Object, eventtype, reason, message string) {
	c.ctrlEventRecorder.Event(object, eventtype, reason, message)
}

// Eventf is just like Event, but with Sprintf for the message field.
func (c *client) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	c.ctrlEventRecorder.Eventf(object, eventtype, reason, messageFmt, args...)
}

// AnnotatedEventf is just like eventf, but with annotations attached
func (c *client) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	c.ctrlEventRecorder.AnnotatedEventf(object, annotations, eventtype, reason, messageFmt, args...)
}

// GetRestConfig return Kubernetes rest Config
func (c *client) GetKubeRestConfig() *rest.Config {
	return c.kubeRestConfig
}

// GetKubeInterface return Kubernetes Interface.
// kubernetes.ClientSet impl kubernetes.Interface
func (c *client) GetKubeInterface() kubernetes.Interface {
	return c.kubeInterface
}

// GetDynamicInterface return dynamic Interface.
// dynamic.ClientSet impl dynamic.Interface
func (c *client) GetDynamicInterface() dynamic.Interface {
	return c.dynamicInterface
}

// GetCtrlRtManager return controller-runtime manager object
func (c *client) GetCtrlRtManager() rtmanager.Manager {
	return c.ctrlRtManager
}

// GetCtrlRtCache return controller-runtime cache object
func (c *client) GetCtrlRtCache() rtcache.Cache {
	return c.ctrlRtCache
}

// GetCtrlRtClient return controller-runtime client
func (c *client) GetCtrlRtClient() rtclient.Client {
	return c.ctrlRtClient
}
