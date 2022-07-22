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
	MingleClientMap      map[string]api.MingleClient
	BeforStartHandleList []api.BeforeStartHandle
	l                    sync.Mutex
	ctx                  context.Context
	stopCh               chan struct{}
	started              int32
	buildClientFunc      BuildClientFunc
}

func (mc *multiClient) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&mc.started, 0, 1) {
		return errors.New("multiclient can't repeat start")
	}

	// save ctx, when add new client
	mc.ctx = ctx

	clsList, err := mc.ClusterCfgManager.GetAll()
	if err != nil {
		return fmt.Errorf("Start multiClient get all cluster info failed %+v", err)
	}

	for _, clsInfo := range clsList {
		cli, err := mc.buildClientFunc(clsInfo, mc.Options)
		if err != nil {
			klog.Errorf("build client %s failed: %s (ignore!!!)", clsInfo.GetName(), err.Error())
			// !import ignore err, because one cluster disconnected not affect connected cluster.
			continue
		}
		err = start(mc.ctx, cli, mc.BeforStartHandleList)
		if err != nil {
			klog.Errorf("start cluster %s failed: %s (ignore!!!)", clsInfo.GetName(), err.Error())
			// !import ignore err, because one cluster disconnected not affect connected cluster.
			continue
		}
		if len(mc.MingleClientMap) == 0 {
			mc.MingleClientMap = make(map[string]api.MingleClient)
		}
		mc.MingleClientMap[clsInfo.GetName()] = cli
	}

	go mc.autoRebuild()

	<-ctx.Done()
	close(mc.stopCh)
	return err
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
			klog.Error("start mingle client %s failed %+v", cli.GetClusterCfgInfo().GetName(), err)
		}
	}()

	return nil
}

func (mc *multiClient) autoRebuild() {
	if mc.RebuildInterval <= 0 {
		return
	}

	var err error
	timer := time.NewTicker(mc.RebuildInterval)
	for {
		select {
		case <-timer.C:
			err = mc.Rebuild()
			if err != nil {
				klog.Errorf("Rebuild failed %+v", err)
			}
		case <-mc.stopCh:
		}
	}
}

// Rebuild get clusterconfigurationmanager GetAll and rebuild clusterClientMap
func (mc *multiClient) Rebuild() error {
	if atomic.LoadInt32(&mc.started) == 0 {
		klog.Warningln("MultiClient not started, rebuild failed.")
		return nil
	}

	mc.l.Lock()
	defer mc.l.Unlock()

	newList, err := mc.ClusterCfgManager.GetAll()
	if err != nil {
		return fmt.Errorf("get all cluster info failed %+v", err)
	}

	newCliMap := make(map[string]api.MingleClient, len(newList))
	var change int
	// add and check new cluster
	for _, newClsInfo := range newList {
		// get old cluster info
		oldCli, exist := mc.MingleClientMap[newClsInfo.GetName()]
		if exist &&
			oldCli.GetClusterCfgInfo().GetKubeConfigType() == newClsInfo.GetKubeConfigType() &&
			oldCli.GetClusterCfgInfo().GetKubeConfig() == newClsInfo.GetKubeConfig() &&
			oldCli.GetClusterCfgInfo().GetKubeContext() == newClsInfo.GetKubeContext() {
			// kubetype, kubeconfig, kubecontext not modify
			newCliMap[oldCli.GetClusterCfgInfo().GetName()] = oldCli
			continue
		}

		// build new client
		cli, err := mc.buildClientFunc(newClsInfo, mc.Options)
		if err != nil {
			klog.Error(err)
			continue
		}

		// start new client
		err = start(mc.ctx, cli, mc.BeforStartHandleList)
		if err != nil {
			klog.Error(err)
			continue
		}

		if exist {
			// kubeconfig modify, should stop old client
			oldCli.Stop()
		}

		newCliMap[newClsInfo.GetName()] = cli
		klog.Infof("auto add mingle client %s", newClsInfo.GetName())
		change++
	}

	// remove unexpect cluster
	for name, oldCli := range mc.MingleClientMap {
		if _, ok := newCliMap[name]; !ok {
			change++
			// not exist, should stop
			go func(cli api.MingleClient) {
				klog.Infof("stop mingle client:%s", cli.GetClusterCfgInfo().GetName())
				cli.Stop()
			}(oldCli)
		}
	}

	// not change return direct
	if change < 1 {
		return nil
	}

	mc.MingleClientMap = newCliMap
	return nil
}

func BuildNormalClient(clsInfo api.ClusterCfgInfo, opts *Options) (api.MingleClient, error) {
	return NewMingleClient(clsInfo, opts)
}
