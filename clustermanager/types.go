package clustermanager

import (
	"time"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/configuration"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	defaultSyncPeriod          = time.Minute * 30
	defaultHealthCheckInterval = time.Second * 5
	defaultExecTimeout         = time.Second * 5
	defaultClusterName         = "meta"
	defaultQPS                 = 100
	defaultBurst               = 120

	minExectimeout = time.Millisecond * 100
)

var (
	// ErrClientNotExist client not exist error
	ErrClientNotExist = "cluster %s not exist"
	// ErrClientNotConnected client disconnected
	ErrClientNotConnected = "cluster %s disconnected"
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
	SetKubeRestConfigFnList []api.SetKubeRestConfig
}

// DefaultClusterCfgInfo default clusterCfgInfo
// kubeconfig use default ~/.kube/config or Kubernetes cluster internal config
func DefaultClusterCfgInfo(clusterName string) api.ClusterCfgInfo {
	if clusterName == "" {
		clusterName = defaultClusterName
	}
	return configuration.BuildDefaultClusterCfgInfo(clusterName)
}

// DefaultOptions use default config
// if scheme is empty use default Kubernetes resource
// disable leader
func DefaultOptions(scheme *runtime.Scheme, qps, burst int) *Options {
	if scheme == nil {
		scheme = runtime.NewScheme()
		clientgoscheme.AddToScheme(scheme)
	}
	if qps < 1 {
		qps = defaultQPS
	}
	if burst < 1 {
		burst = defaultBurst
	}

	return &Options{
		Scheme:              scheme,
		LeaderElection:      false,
		SyncPeriod:          defaultSyncPeriod,
		HealthCheckInterval: defaultHealthCheckInterval,
		ExecTimeout:         defaultExecTimeout,
		SetKubeRestConfigFnList: []api.SetKubeRestConfig{
			func(config *rest.Config) {
				config.QPS = float32(qps)
				config.Burst = burst
			},
		},
	}
}
