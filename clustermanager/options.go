package clustermanager

import (
	"time"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/configuration"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	defaultSyncPeriod          = time.Minute * 30
	defaultHealthCheckInterval = time.Second * 5
	defaultExecTimeout         = time.Second * 5
	defaultAutoRebuildInterval = time.Minute * 5

	defaultManagerClusterName  = "symcn-manager"
	defaultKubeconfigNamespace = "default"
	defaultKubeconfigLabel     = map[string]string{}
	defaultKubeconfigDataKey   = "kubeconfig.yaml"
	defaultKubeconfigStatusKey = "status"
	defaultUserAgent           = "symcn-multi-client"
	defaultQPS                 = 100
	defaultBurst               = 120

	minExectimeout = time.Millisecond * 100
)

var (
	// ErrClientNotExist client not exist error
	ErrClientNotExist = "cluster [%s] not exist"
	// ErrClientNotConnected client disconnected
	ErrClientNotConnected = "cluster [%s] disconnected"
)

// Options options
type Options struct {
	Scheme                  *runtime.Scheme
	LeaderElection          bool
	LeaderElectionNamespace string
	LeaderElectionID        string
	SyncPeriod              time.Duration
	HealthCheckInterval     time.Duration
	ExecTimeout             time.Duration
	UserAgent               string
	QPS                     int
	Burst                   int
	SetKubeRestConfigFnList []api.SetKubeRestConfig
}

type MultiClientConfig struct {
	*Options
	RebuildInterval   time.Duration
	ClusterCfgManager api.ClusterConfigurationManager
	BuildClientFunc   BuildClientFunc
}

type completeConfig struct {
	*MultiClientConfig
}

type CompletedConfig struct {
	*completeConfig
}

func NewMultiClientConfig() *MultiClientConfig {
	mcc := &MultiClientConfig{
		Options:         DefaultOptions(),
		RebuildInterval: defaultAutoRebuildInterval,
		BuildClientFunc: BuildNormalClient,
	}

	return mcc
}

func NewProxyMultiClientConfig() *MultiClientConfig {
	mcc := &MultiClientConfig{
		Options:         DefaultOptions(),
		RebuildInterval: defaultAutoRebuildInterval,
		BuildClientFunc: BuildProxyClient,
	}

	return mcc
}

func Complete(mcc *MultiClientConfig) (*CompletedConfig, error) {
	cc := &CompletedConfig{
		completeConfig: &completeConfig{mcc},
	}

	// check scheme
	if cc.MultiClientConfig.Scheme == nil {
		cc.MultiClientConfig.Scheme = runtime.NewScheme()
		clientgoscheme.AddToScheme(cc.MultiClientConfig.Scheme)
	}

	// check cluster configuration manager
	if cc.MultiClientConfig.ClusterCfgManager != nil {
		return cc, nil
	}

	// build default cluster configuration manager with configmap
	cli, err := NewMingleClient(configuration.BuildDefaultClusterCfgInfo(defaultManagerClusterName), DefaultOptions())
	if err != nil {
		return nil, err
	}

	if mcc.BuildClientFunc == nil {
		mcc.BuildClientFunc = BuildNormalClient
	}

	cc.MultiClientConfig.ClusterCfgManager = configuration.NewClusterCfgManagerWithCM(
		cli.GetKubeInterface(),
		defaultKubeconfigNamespace,
		defaultKubeconfigLabel,
		defaultKubeconfigDataKey,
		defaultKubeconfigStatusKey,
	)

	return cc, nil
}

// New build multiclient
func (cc *CompletedConfig) New() (api.MultiMingleClient, error) {
	mc := &multiClient{
		CompletedConfig:      cc,
		MingleClientMap:      map[string]api.MingleClient{},
		BeforStartHandleList: []api.BeforeStartHandle{},
		stopCh:               make(chan struct{}, 0),
		buildClientFunc:      cc.BuildClientFunc,
	}
	return mc, nil
}

// DefaultClusterCfgInfo default clusterCfgInfo
// kubeconfig use default ~/.kube/config or Kubernetes cluster internal config
func DefaultClusterCfgInfo(clusterName string) api.ClusterCfgInfo {
	if clusterName == "" {
		clusterName = defaultManagerClusterName
	}
	return configuration.BuildDefaultClusterCfgInfo(clusterName)
}

// DefaultOptions use default config
// use default Kubernetes resource
// disable leader
func DefaultOptions() *Options {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	return &Options{
		Scheme:              scheme,
		LeaderElection:      false,
		SyncPeriod:          defaultSyncPeriod,
		HealthCheckInterval: defaultHealthCheckInterval,
		ExecTimeout:         defaultExecTimeout,
		UserAgent:           defaultUserAgent,
		QPS:                 defaultQPS,
		Burst:               defaultBurst,
	}
}

// DefaultOptionsWithScheme use default config
// if scheme is empty use default Kubernetes resource
// disable leader
func DefaultOptionsWithScheme(scheme *runtime.Scheme) *Options {
	opt := DefaultOptions()

	if scheme == nil {
		return opt
	}

	// override scheme
	opt.Scheme = scheme
	return opt
}
