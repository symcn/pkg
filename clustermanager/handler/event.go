package handler

import (
	"github.com/symcn/api"
	ktypes "k8s.io/apimachinery/pkg/types"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type transformNamespacedNameEventHandler struct {
	NameFunc      api.ObjectTransformFunc
	NamespaceFunc api.ObjectTransformFunc
}

// NewDefaultTransformNamespacedNameEventHandler build transform namespace and name eventHandler
func NewDefaultTransformNamespacedNameEventHandler() api.EventHandler {
	return &transformNamespacedNameEventHandler{
		NameFunc: func(obj rtclient.Object) string {
			return obj.GetName()
		},
		NamespaceFunc: func(obj rtclient.Object) string {
			return obj.GetNamespace()
		},
	}
}

func (t *transformNamespacedNameEventHandler) Create(obj rtclient.Object, queue api.WorkQueue) {
	queue.Add(ktypes.NamespacedName{
		Name:      t.NameFunc(obj),
		Namespace: t.NamespaceFunc(obj),
	})
}

func (t *transformNamespacedNameEventHandler) Update(oldObj, newObj rtclient.Object, queue api.WorkQueue) {
	queue.Add(ktypes.NamespacedName{
		Name:      t.NameFunc(newObj),
		Namespace: t.NamespaceFunc(newObj),
	})
}

func (t *transformNamespacedNameEventHandler) Delete(obj rtclient.Object, queue api.WorkQueue) {
	queue.Add(ktypes.NamespacedName{
		Name:      t.NameFunc(obj),
		Namespace: t.NamespaceFunc(obj),
	})
}

func (t *transformNamespacedNameEventHandler) Generic(obj rtclient.Object, queue api.WorkQueue) {
	queue.Add(ktypes.NamespacedName{
		Name:      t.NameFunc(obj),
		Namespace: t.NamespaceFunc(obj),
	})
}
