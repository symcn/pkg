package clustermanager

import (
	"context"
	"errors"

	multicluster "github.com/oam-dev/cluster-gateway/pkg/apis/cluster/transport"
	"github.com/symcn/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	rtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	rtmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

type proxyGatewayClient struct {
	*Options

	clusterCfg     api.ClusterCfgInfo
	stopCh         chan struct{}
	connected      bool
	started        bool
	internalCancel context.CancelFunc
	informerList   []rtcache.Informer

	kubeRestConfig *rest.Config
	kubeInterface  kubernetes.Interface

	ctrlRtManager     rtmanager.Manager
	ctrlRtCache       rtcache.Cache
	ctrlRtClient      rtclient.Client
	ctrlEventRecorder record.EventRecorder
}

func NewProxyGatewayMingleClient(clusterCfg api.ClusterCfgInfo, opt *Options) (api.MingleClient, error) {
	proxyClient := &proxyGatewayClient{
		Options:    opt,
		clusterCfg: clusterCfg,
	}

	// 1. pre check
	if err := proxyClient.preCheck(); err != nil {
		return nil, err
	}

	// 2. transport to client
	cli := &client{
		Options:      proxyClient.Options,
		clusterCfg:   proxyClient.clusterCfg,
		stopCh:       make(chan struct{}, 0),
		informerList: []rtcache.Informer{},
	}

	if err := cli.initialization(); err != nil {
		return nil, err
	}

	return cli, nil
}

func (p *proxyGatewayClient) preCheck() error {
	if p.Options == nil {
		return errors.New("options is empty")
	}

	if p.clusterCfg.GetName() == "" || p.clusterCfg.GetKubeConfigType() != api.KubeConfigTypeInCluster {
		return errors.New("cluster name is empty or kubeconfig type not InCluster")
	}

	// cluster scheme must not empty
	if p.Options.Scheme == nil {
		return errors.New("scheme is empty")
	}

	// exectimeout check
	if p.Options.ExecTimeout < minExectimeout {
		klog.Warningf("exectimeout should lager than 100ms, too small will return timeout mostly, use default %v", defaultExecTimeout)
		p.Options.ExecTimeout = defaultExecTimeout
	}

	// set QPS and Burst
	if p.Options.QPS > 0 && p.Options.Burst > 0 {
		if len(p.SetKubeRestConfigFnList) == 0 {
			p.SetKubeRestConfigFnList = []api.SetKubeRestConfig{}
		}
		klog.Infof("cluster %s connection use QPS %d and Burst %d", p.clusterCfg.GetName(), p.QPS, p.Burst)
		p.Options.SetKubeRestConfigFnList = append(p.Options.SetKubeRestConfigFnList, func(config *rest.Config) {
			config.QPS = float32(p.QPS)
			config.Burst = p.Burst
			config.UserAgent = p.UserAgent
		})
	}

	// import: add roundtripper
	p.Options.SetKubeRestConfigFnList = append(p.Options.SetKubeRestConfigFnList, func(config *rest.Config) {
		config.Wrap(multicluster.NewProxyPathPrependingClusterGatewayRoundTripper(p.clusterCfg.GetName()).NewRoundTripper)
	})

	return nil
}
