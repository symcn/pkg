package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/symcn/api"
	"k8s.io/klog/v2"
)

type BuildClientFunc func(api.ClusterCfgInfo, *Options) (api.MingleClient, error)

type multiClient struct {
	*CompletedConfig
	MingleClientMap         map[string]api.MingleClient
	BeforStartHandleList    []api.BeforeStartHandle
	l                       sync.Mutex
	ctx                     context.Context
	stopCh                  chan struct{}
	started                 int32
	buildClientFunc         BuildClientFunc
	clusterEventHandlerList []api.ClusterEventHandler
}

func (mc *multiClient) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&mc.started, 0, 1) {
		return errors.New("multiclient can't repeat start")
	}

	// save ctx, when add new client
	mc.ctx = ctx

	if err := mc.loopFetchClient(); err != nil {
		return err
	}

	<-ctx.Done()
	mc.clean()
	return nil
}

func (mc *multiClient) loopFetchClient() error {
	err := mc.FetchClientInfoOnce()
	if err != nil {
		return err
	}

	if mc.FetchInterval <= 0 {
		return nil
	}
	go func() {
		var err error
		timer := time.NewTicker(mc.FetchInterval)
		for {
			select {
			case <-timer.C:
				err = mc.FetchClientInfoOnce()
				if err != nil {
					klog.ErrorS(err, "FetchClientInfoOnce failed")
				}
			case <-mc.stopCh:
			}
		}
	}()
	return nil
}

func (mc *multiClient) clean() {
	close(mc.stopCh)
}

// AddClusterEventHandler implements api.MultiMingleClient
func (mc *multiClient) AddClusterEventHandler(handler api.ClusterEventHandler) {
	mc.l.Lock()
	defer mc.l.Unlock()

	// ignore multi client start already, and AddClusterEventHandler invoke get empty client.
	for _, cli := range mc.MingleClientMap {
		handler.OnAdd(mc.ctx, cli)
	}
	mc.clusterEventHandlerList = append(mc.clusterEventHandlerList, handler)
}

// FetchClientInfoOnce get clusterconfigurationmanager GetAll and rebuild clusterClientMap
func (mc *multiClient) FetchClientInfoOnce() error {
	if atomic.LoadInt32(&mc.started) == 0 {
		klog.Warningln("MultiClient not started, rebuild failed.")
		return nil
	}

	mc.l.Lock()
	defer mc.l.Unlock()

	freshList, err := mc.ClusterCfgManager.GetAll()
	if err != nil {
		return fmt.Errorf("get all cluster info failed %+v", err)
	}

	freshCliMap := make(map[string]api.MingleClient, len(freshList))
	var change int
	// add and check new cluster
	for _, freshClsInfo := range freshList {
		// get old cluster info
		currentCli, exist := mc.MingleClientMap[freshClsInfo.GetName()]
		if exist &&
			currentCli.GetClusterCfgInfo().GetKubeConfigType() == freshClsInfo.GetKubeConfigType() &&
			currentCli.GetClusterCfgInfo().GetKubeConfig() == freshClsInfo.GetKubeConfig() &&
			currentCli.GetClusterCfgInfo().GetKubeContext() == freshClsInfo.GetKubeContext() {
			// kubetype, kubeconfig, kubecontext not modify
			freshCliMap[currentCli.GetClusterCfgInfo().GetName()] = currentCli
			continue
		}

		cli, err := mc.buildNewCluster(freshClsInfo, mc.Options)
		if err != nil {
			// !import ignore err, because one cluster disconnected not affect connected cluster.
			klog.ErrorS(err, "buildNewCluster failed (ignore!!!).", "clusterName", freshClsInfo.GetName())
			continue
		}

		if exist {
			// kubeconfig modify, should stop old client
			klog.InfoS("Configuration modified, stop old mingle client", "clusterName", cli.GetClusterCfgInfo().GetName())
			mc.stopCluster(currentCli)
		}

		freshCliMap[freshClsInfo.GetName()] = cli
		klog.InfoS("Auto add mingle client successful!", "clusterName", freshClsInfo.GetName())
		change++
	}

	// remove unexpect cluster
	for name, currentCli := range mc.MingleClientMap {
		if _, ok := freshCliMap[name]; !ok {
			change++
			// not exist, should stop
			go func(cli api.MingleClient) {
				klog.InfoS("Stop mingle client", "clusterName", cli.GetClusterCfgInfo().GetName())
				mc.stopCluster(cli)
			}(currentCli)
		}
	}

	// client list changed.
	if change > 0 {
		mc.MingleClientMap = freshCliMap
	}
	return nil
}

func (mc *multiClient) buildNewCluster(newClsInfo api.ClusterCfgInfo, options *Options) (api.MingleClient, error) {
	// build new client
	cli, err := mc.buildClientFunc(newClsInfo, options)
	if err != nil {
		return nil, err
	}

	// start new client
	err = start(mc.ctx, cli, mc.BeforStartHandleList)
	if err != nil {
		// clear client resources
		cli.Stop()
		return nil, err
	}

	if len(mc.clusterEventHandlerList) > 0 {
		for _, handler := range mc.clusterEventHandlerList {
			handler.OnAdd(mc.ctx, cli)
		}
	}

	return cli, nil
}

func start(ctx context.Context, cli api.MingleClient, beforStartHandleList []api.BeforeStartHandle) error {
	var err error
	for _, handler := range beforStartHandleList {
		err = handler(ctx, cli)
		if err != nil {
			return fmt.Errorf("invoke mingle client %s BeforeHandle failed %+v", cli.GetClusterCfgInfo().GetName(), err)
		}
	}

	go func() {
		err = cli.Start(ctx)
		if err != nil {
			klog.ErrorS(err, "start mingle client failed", "clusterName", cli.GetClusterCfgInfo().GetName())
		}
	}()

	return nil
}

func (mc *multiClient) stopCluster(cli api.MingleClient) {
	if len(mc.clusterEventHandlerList) > 0 {
		for _, handler := range mc.clusterEventHandlerList {
			handler.OnDelete(mc.ctx, cli)
		}
	}
	cli.Stop()
}

func BuildNormalClient(clsInfo api.ClusterCfgInfo, opts *Options) (api.MingleClient, error) {
	return NewMingleClient(clsInfo, opts)
}
