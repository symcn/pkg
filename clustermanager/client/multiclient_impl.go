package client

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/handler"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddResourceEventHandler loop each mingleclient invoke AddResourceEventHandler
func (mc *multiClient) AddResourceEventHandler(obj rtclient.Object, handler cache.ResourceEventHandler) error {
	mc.l.Lock()
	defer mc.l.Unlock()

	mc.RegistryBeforeStartHandler(func(ctx context.Context, cli api.MingleClient) error {
		err := cli.AddResourceEventHandler(obj, handler)
		if err != nil {
			return fmt.Errorf("cluster %s AddResourceEventHandler failed %+v", cli.GetClusterCfgInfo().GetName(), err)
		}
		return nil
	})
	return nil
}

// TriggerSync just trigger each mingleclient cache resource without handler
func (mc *multiClient) TriggerSync(obj rtclient.Object) error {
	mc.l.Lock()
	defer mc.l.Unlock()

	mc.RegistryBeforeStartHandler(func(ctx context.Context, cli api.MingleClient) error {
		_, err := cli.GetInformer(obj)
		if err != nil {
			return fmt.Errorf("cluster %s TriggerSync failed %+v", cli.GetClusterCfgInfo().GetName(), err)
		}
		return nil
	})
	return nil
}

// SetIndexField loop each mingleclient invoke SetIndexField
func (mc *multiClient) SetIndexField(obj rtclient.Object, field string, extractValue rtclient.IndexerFunc) error {
	mc.l.Lock()
	defer mc.l.Unlock()

	mc.RegistryBeforeStartHandler(func(ctx context.Context, cli api.MingleClient) error {
		err := cli.SetIndexField(obj, field, extractValue)
		if err != nil {
			return fmt.Errorf("cluster %s SetIndexField failed %+v", cli.GetClusterCfgInfo().GetName(), err)
		}
		return nil
	})
	return nil
}

// Watch takes events provided by a Source and uses the EventHandler to
// enqueue reconcile.Requests in response to the events.
//
// Watch may be provided one or more Predicates to filter events before
// they are given to the EventHandler.  Events will be passed to the
// EventHandler if all provided Predicates evaluate to true.
func (mc *multiClient) Watch(obj rtclient.Object, queue api.WorkQueue, evtHandler api.EventHandler, predicates ...api.Predicate) error {
	if queue == nil {
		return errors.New("api.WorkQueue is nil")
	}
	err := mc.AddResourceEventHandler(obj, handler.NewResourceEventHandler(queue, evtHandler, predicates...))
	if err != nil {
		return fmt.Errorf("Watch resource failed %+v", err)
	}
	return nil
}

// HasSynced return true if all mingleclient and all informers underlying store has synced
// !import if informerlist is empty, will return true
func (mc *multiClient) HasSynced() bool {
	if atomic.LoadInt32(&mc.started) == 0 {
		klog.Warningln("MultiClient not start, HasSynced return false.")
		return false
	}

	mc.l.Lock()
	defer mc.l.Unlock()

	for _, cli := range mc.MingleClientMap {
		if !cli.HasSynced() {
			return false
		}
	}
	return true
}

// GetWithName returns MingleClient object with name
func (mc *multiClient) GetWithName(name string) (api.MingleClient, error) {
	mc.l.Lock()
	defer mc.l.Unlock()

	if cli, ok := mc.MingleClientMap[name]; ok {
		return cli, nil
	}
	return nil, fmt.Errorf(ErrClientNotExist, name)
}

// GetConnectedWithName returns MingleClient object with name and status is connected
func (mc *multiClient) GetConnectedWithName(name string) (api.MingleClient, error) {
	mc.l.Lock()
	defer mc.l.Unlock()

	if cli, ok := mc.MingleClientMap[name]; ok {
		if cli.IsConnected() {
			return cli, nil
		}
		return nil, fmt.Errorf(ErrClientNotConnected, name)
	}
	return nil, fmt.Errorf(ErrClientNotExist, name)
}

// GetAll returns all MingleClient
func (mc *multiClient) GetAll() []api.MingleClient {
	mc.l.Lock()
	defer mc.l.Unlock()

	list := make([]api.MingleClient, 0, len(mc.MingleClientMap))
	for _, cli := range mc.MingleClientMap {
		list = append(list, cli)
	}
	return list
}

// GetAllConnected returns all MingleClient which status is connected
func (mc *multiClient) GetAllConnected() []api.MingleClient {
	mc.l.Lock()
	defer mc.l.Unlock()

	list := make([]api.MingleClient, 0, len(mc.MingleClientMap))
	for _, cli := range mc.MingleClientMap {
		if cli.IsConnected() {
			list = append(list, cli)
		}
	}
	return list
}

// RegistryBeforeStartHandler registry BeforeStartHandle
func (mc *multiClient) RegistryBeforeStartHandler(handler api.BeforeStartHandle) {
	mc.BeforStartHandleList = append(mc.BeforStartHandleList, handler)
}
