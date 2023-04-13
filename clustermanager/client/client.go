package client

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/symcn/api"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	controllers "sigs.k8s.io/controller-runtime"
	rtcache "sigs.k8s.io/controller-runtime/pkg/cache"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	rtmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

type client struct {
	*Options

	clusterCfg     api.ClusterCfgInfo
	stopCh         chan struct{}
	connected      bool
	started        int32
	internalCancel context.CancelFunc
	informerList   []rtcache.Informer

	kubeRestConfig   *rest.Config
	kubeInterface    kubernetes.Interface
	dynamicInterface dynamic.Interface

	ctrlRtManager     rtmanager.Manager
	ctrlRtCache       rtcache.Cache
	ctrlRtClient      rtclient.Client
	ctrlEventRecorder record.EventRecorder
}

// NewMingleClient build api.MingleClient
func NewMingleClient(clusterCfg api.ClusterCfgInfo, opt *Options) (api.MingleClient, error) {
	cli := &client{
		Options:      opt,
		clusterCfg:   clusterCfg,
		stopCh:       make(chan struct{}),
		informerList: []rtcache.Informer{},
	}

	// 1. pre check
	if err := cli.preCheck(); err != nil {
		return nil, err
	}

	// 2. initialization
	if err := cli.initialization(); err != nil {
		return nil, err
	}

	return cli, nil
}
func (c *client) preCheck() error {
	if c.Options == nil {
		return errors.New("options is empty")
	}

	// clusterconfig and cluster name must not empty
	if c.clusterCfg == nil || c.clusterCfg.GetName() == "" {
		return errors.New("cluster info is empty or cluster name is empty")
	}

	// cluster scheme must not empty
	if c.Options.Scheme == nil {
		return errors.New("scheme is empty")
	}

	// exectimeout check
	if c.Options.ExecTimeout < minExectimeout {
		klog.Warningf("exectimeout should lager than 100ms, too small will return timeout mostly, use default %v", defaultExecTimeout)
		c.Options.ExecTimeout = defaultExecTimeout
	}

	// set QPS and Burst
	if c.QPS > 0 && c.Burst > 0 {
		if len(c.SetKubeRestConfigFnList) == 0 {
			c.SetKubeRestConfigFnList = []api.SetKubeRestConfig{}
		}
		klog.Infof("cluster %s connection use QPS %d and Burst %d", c.clusterCfg.GetName(), c.QPS, c.Burst)
		c.SetKubeRestConfigFnList = append(c.SetKubeRestConfigFnList, func(config *rest.Config) {
			config.QPS = float32(c.QPS)
			config.Burst = c.Burst
			config.UserAgent = c.UserAgent
		})
	}

	return nil
}

func (c *client) initialization() error {
	var err error
	// Step 1. build restconfig
	c.kubeRestConfig, err = buildClientCmd(c.clusterCfg, c.SetKubeRestConfigFnList)
	if err != nil {
		return fmt.Errorf("cluster %s build kubernetes failed %+v", c.clusterCfg.GetName(), err)
	}

	// Step 2. build kubernetes interface
	c.kubeInterface, err = kubernetes.NewForConfig(c.kubeRestConfig)
	if err != nil {
		return fmt.Errorf("cluster %s build kubernetes interface failed %+v", c.clusterCfg.GetName(), err)
	}

	// Step 3. build dynamic interface
	c.dynamicInterface, err = dynamic.NewForConfig(c.kubeRestConfig)
	if err != nil {
		return fmt.Errorf("cluster %s build dynamic interface failed %+v", c.clusterCfg.GetName(), err)
	}

	// Step 4. build controller-runtime manager
	c.ctrlRtManager, err = controllers.NewManager(c.kubeRestConfig, rtmanager.Options{
		Scheme:                  c.Scheme,
		SyncPeriod:              &c.SyncPeriod,
		LeaderElection:          c.LeaderElection,
		LeaderElectionNamespace: c.LeaderElectionNamespace,
		LeaderElectionID:        c.LeaderElectionID,
		MetricsBindAddress:      "0",
		HealthProbeBindAddress:  "0",

		// webhook configuration
		// TODO: expose most field.
		Port:    c.Port,
		Host:    c.Host,
		CertDir: c.CertDir,
		TLSOpts: c.TLSOpts,
	})
	if err != nil {
		return fmt.Errorf("cluster %s build controller-runtime manager failed: %+v", c.clusterCfg.GetName(), err)
	}
	c.ctrlRtClient = c.ctrlRtManager.GetClient()
	c.ctrlRtCache = c.ctrlRtManager.GetCache()
	c.ctrlEventRecorder = c.ctrlRtManager.GetEventRecorderFor(c.clusterCfg.GetName())

	return nil
}

func (c *client) autoHealthCheck() {
	clusterHealthCheckOnce := func() {
		ok, err := healthRequestWithTimeout(c.kubeInterface.Discovery().RESTClient(), c.ExecTimeout)
		if err != nil {
			klog.Errorf("cluster %s check healthy failed %+v", c.clusterCfg.GetName(), err)
		}
		c.connected = ok
	}

	// first check
	clusterHealthCheckOnce()

	// it will pointless when interval less than 1s
	if c.HealthCheckInterval < time.Second {
		klog.Warningf("cluster %s not enabled healthy check, interval must be greater than 1s")
		return
	}

	timer := time.NewTicker(c.Options.HealthCheckInterval)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			clusterHealthCheckOnce()
		case <-c.stopCh:
			return
		}
	}
}

// Start client and blocks until the context is cancelled
// Returns an error if there is an error starting
func (c *client) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return fmt.Errorf("client %s can't repeat start", c.clusterCfg.GetName())
	}

	var err error
	ctx, c.internalCancel = context.WithCancel(ctx)

	go func() {
		err = c.ctrlRtManager.Start(ctx)
		if err != nil {
			klog.Errorf("start cluster %s error: %+v", c.clusterCfg.GetName(), err)
			close(c.stopCh)
		}
		klog.Warningf("cluster %s stoped.", c.clusterCfg.GetName())
	}()

	// health check
	go c.autoHealthCheck()

	select {
	case <-ctx.Done():
		return err
	case <-c.stopCh:
		c.Stop()
		return err
	}
}

// Stop stop mingle client, just use with multiclient, not recommend use direct
func (c *client) Stop() {
	if c.internalCancel == nil {
		return
	}
	c.internalCancel()
}

// IsConnected return connected status
func (c *client) IsConnected() bool {
	return c.connected
}

// GetClusterCfgInfo returns cluster configuration info
func (c *client) GetClusterCfgInfo() api.ClusterCfgInfo {
	return c.clusterCfg
}
