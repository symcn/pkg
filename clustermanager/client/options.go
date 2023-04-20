package client

import (
	"crypto/tls"
	"time"

	"github.com/go-logr/logr"
	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/configuration"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	defaultSyncPeriod          = time.Minute * 30
	defaultHealthCheckInterval = time.Second * 5
	defaultExecTimeout         = time.Second * 5
	defaultAutoFetchInterval   = time.Minute * 5

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
	WebhookOptions

	Scheme                  *runtime.Scheme
	Logger                  logr.Logger
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

// WebhookOptions webhook configuration for controller-manager
type WebhookOptions struct {
	// Port is the port that the webhook server serves at.
	// It is used to set webhook.Server.Port if WebhookServer is not set.
	Port int
	// Host is the hostname that the webhook server binds to.
	// It is used to set webhook.Server.Host if WebhookServer is not set.
	Host string

	// CertDir is the directory that contains the server key and certificate.
	// If not set, webhook server would look up the server key and certificate in
	// {TempDir}/k8s-webhook-server/serving-certs. The server key and certificate
	// must be named tls.key and tls.crt, respectively.
	// It is used to set webhook.Server.CertDir if WebhookServer is not set.
	CertDir string

	// TLSOpts is used to allow configuring the TLS config used for the webhook server.
	TLSOpts []func(*tls.Config)
}

type MultiClientConfig struct {
	*Options
	FetchInterval     time.Duration
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
		FetchInterval:   defaultAutoFetchInterval,
		BuildClientFunc: BuildNormalClient,
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
		stopCh:               make(chan struct{}),
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
