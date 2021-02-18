package clustermanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/symcn/api"
	"k8s.io/klog/v2"
)

type multiclient struct {
	*Options
	clusterCfgManager    api.ClusterConfigurationManager
	rebuildInterval      time.Duration
	l                    sync.Mutex
	ctx                  context.Context
	stopCh               chan struct{}
	started              bool
	mingleClientMap      map[string]api.MingleClient
	beforStartHandleList []api.BeforeStartHandle
}

// NewMultiMingleClient build multiclient
func NewMultiMingleClient(clusterCfgManager api.ClusterConfigurationManager, rebuildInterval time.Duration, opt *Options) (api.MultiMingleClient, error) {
	multiCli := &multiclient{
		Options:              opt,
		clusterCfgManager:    clusterCfgManager,
		rebuildInterval:      rebuildInterval,
		stopCh:               make(chan struct{}, 0),
		mingleClientMap:      map[string]api.MingleClient{},
		beforStartHandleList: []api.BeforeStartHandle{},
	}

	clsList, err := multiCli.clusterCfgManager.GetAll()
	if err != nil {
		return nil, fmt.Errorf("NewMulticMingleClient get all cluster info failed %+v", err)
	}
	for _, clsInfo := range clsList {
		cli, err := multiCli.buildClient(clsInfo)
		if err != nil {
			return nil, err
		}
		multiCli.mingleClientMap[clsInfo.GetName()] = cli
	}
	return multiCli, nil
}

func (mc *multiclient) Start(ctx context.Context) error {
	if mc.started {
		return errors.New("multiclient can't repeat start")
	}
	mc.started = true
	// save ctx, when add new client
	mc.ctx = ctx

	mc.l.Lock()
	var err error
	for _, cli := range mc.mingleClientMap {
		err = start(mc.ctx, cli, mc.beforStartHandleList)
		if err != nil {
			mc.l.Unlock()
			return err
		}
	}
	mc.l.Unlock()

	go mc.autoRebuild()

	select {
	case <-ctx.Done():
		close(mc.stopCh)
		return err
	}
}

func start(ctx context.Context, cli api.MingleClient, beforStartHandleList []api.BeforeStartHandle) error {
	var err error
	for _, handler := range beforStartHandleList {
		err = handler(cli)
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

func (mc *multiclient) autoRebuild() {
	if mc.rebuildInterval <= 0 {
		return
	}

	var err error
	timer := time.NewTicker(mc.rebuildInterval)
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
func (mc *multiclient) Rebuild() error {
	if !mc.started {
		return nil
	}

	mc.l.Lock()
	defer mc.l.Unlock()

	newList, err := mc.clusterCfgManager.GetAll()
	if err != nil {
		return fmt.Errorf("get all cluster info failed %+v", err)
	}

	newCliMap := make(map[string]api.MingleClient, len(newList))
	var change int
	// add and check new cluster
	for _, newClsInfo := range newList {
		// get old cluster info
		oldCli, exist := mc.mingleClientMap[newClsInfo.GetName()]
		if exist &&
			oldCli.GetClusterCfgInfo().GetKubeConfigType() == newClsInfo.GetKubeConfigType() &&
			oldCli.GetClusterCfgInfo().GetKubeConfig() == newClsInfo.GetKubeConfig() &&
			oldCli.GetClusterCfgInfo().GetKubeContext() == newClsInfo.GetKubeContext() {
			// kubetype, kubeconfig, kubecontext not modify
			newCliMap[oldCli.GetClusterCfgInfo().GetName()] = oldCli
			continue
		}

		// build new client
		cli, err := mc.buildClient(newClsInfo)
		if err != nil {
			klog.Error(err)
			continue
		}

		// start new client
		err = start(mc.ctx, cli, mc.beforStartHandleList)
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
	for name, oldCli := range mc.mingleClientMap {
		if _, ok := newCliMap[name]; !ok {
			change++
			// not exist, should stop
			go func(cli api.MingleClient) {
				cli.Stop()
			}(oldCli)
		}
	}

	// not change return direct
	if change < 1 {
		return nil
	}

	mc.mingleClientMap = newCliMap
	return nil
}

func (mc *multiclient) buildClient(clsInfo api.ClusterCfgInfo) (api.MingleClient, error) {
	return NewMingleClient(clsInfo, mc.Options)
}
