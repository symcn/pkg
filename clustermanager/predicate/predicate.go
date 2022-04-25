package predicate

import (
	"strings"

	"github.com/symcn/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type generationChangedPredicate struct {
	Funcs
}

func (g *generationChangedPredicate) Update(oldObj, newObj rtclient.Object) bool {
	if oldObj == nil {
		klog.Error("Update event has no old object to update")
		return false
	}
	if newObj == nil {
		klog.Error("Update event has no new object to update.")
		return false
	}
	return oldObj.GetGeneration() != newObj.GetGeneration()
}

func NewGengerationChangedPredicate() api.Predicate {
	return &generationChangedPredicate{}
}

// NamespacePredicate filter namespace
func NamespacePredicate(nslist ...string) api.Predicate {
	return &base{
		handler: func(obj client.Object) bool {
			for _, ns := range nslist {
				if ns == "*" || strings.EqualFold(ns, obj.GetNamespace()) {
					return true
				}
			}
			return false
		},
	}
}

// LabelsKeyPredicate filter labels key not exists
func LabelsKeyPredicate(keys ...string) api.Predicate {
	return &base{
		handler: func(obj client.Object) bool {
			if len(obj.GetLabels()) == 0 {
				return false
			}
			for _, key := range keys {
				if _, ok := obj.GetLabels()[key]; !ok {
					return false
				}
			}
			return true
		},
	}
}
