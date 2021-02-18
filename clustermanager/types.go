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
	QPS                     int
	Burst                   int
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
