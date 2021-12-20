package client

import (
	"errors"
	"fmt"

	multicluster "github.com/oam-dev/cluster-gateway/pkg/apis/cluster/transport"
	"github.com/symcn/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type proxyClient struct {
	clusterCfg api.ClusterCfgInfo

	scheme           *runtime.Scheme
	kubeRestConfig   *rest.Config
	kubeInterface    kubernetes.Interface
	dynamicInterface dynamic.Interface
	runtimeInterface rtclient.Client
}

func NewProxyGatewayMingleClient(clusterCfg api.ClusterCfgInfo, scheme *runtime.Scheme) (api.MingleProxyClient, error) {
	pcli := &proxyClient{
		clusterCfg: clusterCfg,
		scheme:     scheme,
	}

	// 1. pre check
	if err := pcli.preCheck(); err != nil {
		return nil, err
	}

	// 3. initialization
	if err := pcli.initialization(); err != nil {
		return nil, err
	}
	return pcli, nil
}

func (pc *proxyClient) preCheck() error {
	// clusterconfig and cluster name must not empty
	if pc.clusterCfg == nil || pc.clusterCfg.GetName() == "" {
		return errors.New("proxy cluster info is empty or cluster name is empty")
	}
	return nil
}

func (pc *proxyClient) initialization() error {
	var err error
	// Step 1. build restconfig
	pc.kubeRestConfig, err = buildClientCmd(pc.clusterCfg, nil)
	if err != nil {
		return fmt.Errorf("proxy cluster %s build kubernetes failed %+v", pc.clusterCfg.GetName(), err)
	}
	pc.kubeRestConfig.Wrap(multicluster.NewProxyPathPrependingClusterGatewayRoundTripper(pc.clusterCfg.GetName()).NewRoundTripper)

	// Step 2. build kubernetes interface
	pc.kubeInterface, err = kubernetes.NewForConfig(pc.kubeRestConfig)
	if err != nil {
		return fmt.Errorf("proxy cluster %s build kubernetes interface failed %+v", pc.clusterCfg.GetName(), err)
	}

	// Step 3. build dynamic interface
	pc.dynamicInterface, err = dynamic.NewForConfig(pc.kubeRestConfig)
	if err != nil {
		return fmt.Errorf("proxy cluster %s build dynamic interface failed %+v", pc.clusterCfg.GetName(), err)
	}

	// // Step 4. build runtime client use lazy load
	// pc.runtimeInterface, err = rtclient.New(pc.kubeRestConfig, rtclient.Options{})
	// if err != nil {
	//     return fmt.Errorf("proxy cluster %s build runtime client failed %+v", pc.clusterCfg.GetName(), err)
	// }

	return nil
}

// GetRestConfig return Kubernetes rest Config
func (pc *proxyClient) GetKubeRestConfig() *rest.Config {
	return pc.kubeRestConfig
}

// GetKubeInterface return Kubernetes Interface.
// kubernetes.ClientSet impl kubernetes.Interface
func (pc *proxyClient) GetKubeInterface() kubernetes.Interface {
	return pc.kubeInterface
}

// GetDynamicInterface return dynamic Interface.
// dynamic.ClientSet impl dynamic.Interface
func (pc *proxyClient) GetDynamicInterface() dynamic.Interface {
	return pc.dynamicInterface
}

// GetRuntimeClient() return controller runtime client
func (pc *proxyClient) GetRuntimeClient() rtclient.Client {
	if pc.runtimeInterface == nil {
		var err error
		pc.runtimeInterface, err = rtclient.New(pc.kubeRestConfig, rtclient.Options{Scheme: pc.scheme})
		if err != nil {
			panic(fmt.Errorf("proxy cluster %s build runtime client failed %+v", pc.clusterCfg.GetName(), err))
		}
	}
	return pc.runtimeInterface
}

// GetClusterCfgInfo returns cluster configuration info
func (pc *proxyClient) GetClusterCfgInfo() api.ClusterCfgInfo {
	return pc.clusterCfg
}
