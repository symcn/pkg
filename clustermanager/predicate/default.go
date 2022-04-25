package predicate

import rtclient "sigs.k8s.io/controller-runtime/pkg/client"

type Funcs struct {
	// Create returns true if the Create event should be processed
	CreateFunc func(obj rtclient.Object) bool

	// Delete returns true if the Delete event should be processed
	DeleteFunc func(obj rtclient.Object) bool

	// Update returns true if the Update event should be processed
	UpdateFunc func(oldObj, newObj rtclient.Object) bool

	// Generic returns true if the Generic event should be processed
	GenericFunc func(obj rtclient.Object) bool
}

// Create returns true if the Create event should be processed
func (f *Funcs) Create(obj rtclient.Object) bool {
	if f.CreateFunc != nil {
		return f.CreateFunc(obj)
	}
	return true
}

// Delete returns true if the Delete event should be processed
func (f *Funcs) Delete(obj rtclient.Object) bool {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(obj)
	}
	return true
}

// Update returns true if the Update event should be processed
func (f *Funcs) Update(oldObj, newObj rtclient.Object) bool {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(newObj, oldObj)
	}
	return true
}

// Generic returns true if the Generic event should be processed
func (f *Funcs) Generic(obj rtclient.Object) bool {
	if f.GenericFunc != nil {
		return f.GenericFunc(obj)
	}
	return true
}
