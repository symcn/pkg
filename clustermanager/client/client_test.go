package client

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestExceptionNewMingleClient(t *testing.T) {
	// precheck
	t.Run("config is empty", func(t *testing.T) {
		_, err := NewMingleClient(nil, nil)
		if err == nil {
			t.Error("client config is empty, should be error")
		}
	})
	t.Run("cluster configuration is empty", func(t *testing.T) {
		_, err := NewMingleClient(nil, nil)
		if err == nil {
			t.Error("cluster configuration config is empty, should be error")
		}
	})
	t.Run("scheme is empty", func(t *testing.T) {
		cfg := DefaultClusterCfgInfo("")
		opt := DefaultOptions()
		opt.Scheme = nil
		_, err := NewMingleClient(cfg, opt)
		if err == nil {
			t.Error("scheme config is empty, should be error")
		}
	})
	t.Run("exectimeout to small", func(t *testing.T) {
		cfg := DefaultClusterCfgInfo("")
		opt := DefaultOptions()
		opt.ExecTimeout = time.Millisecond * 10
		_, err := NewMingleClient(cfg, opt)
		if err != nil {
			t.Error(err)
		}
	})

	// health check
	t.Run("no health check", func(t *testing.T) {
		cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		go func() {
			err = cli.Start(ctx)
		}()

		time.Sleep(time.Millisecond * 100)
		cancel()
		time.Sleep(time.Millisecond * 100)

		if err != nil {
			t.Error(err)
		}
	})

	// start
	t.Run("repeat start", func(t *testing.T) {
		cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
		if err != nil {
			t.Error(err)
			return
		}
		errCh := make(chan error, 2)
		ctx, cancel := context.WithCancel(context.TODO())
		go func() {
			errCh <- cli.Start(ctx)
		}()
		go func() {
			errCh <- cli.Start(ctx)
		}()
		defer cancel()

		for i := 0; i < 2; i++ {
			err = <-errCh
			if err != nil {
				return
			}
		}
		// exec this means multi Start without err
		t.Log("repeat start should err")
	})

	t.Run("stop", func(t *testing.T) {
		cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		go func() {
			err = cli.Start(ctx)
			if err != nil {
				t.Error(err)
			}
		}()
		defer cancel()
		// maybe internalCancel is nil
		cli.Stop()
		time.Sleep(time.Millisecond * 100)
		cli.Stop()
	})

	t.Run("start connect status", func(t *testing.T) {
		cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		go func() {
			err = cli.Start(ctx)
			if err != nil {
				t.Error(err)
			}
		}()

		time.Sleep(time.Millisecond * 100)
		home, _ := os.UserHomeDir()
		path := home + "/.kube/config"
		_, err = os.Stat(path)
		if err == nil {
			if !cli.IsConnected() {
				t.Error("exist kubeconfig should connected Kubernetes cluster")
			}
		}
	})

	t.Run("repeat watch", func(t *testing.T) {
		cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
		if err != nil {
			t.Error(err)
			return
		}
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*2)
		defer cancel()

		cli.AddResourceEventHandler(&corev1.Pod{}, cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				t.Log("1")
			},
		})
		cli.AddResourceEventHandler(&corev1.Pod{}, cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				t.Log("2")
			},
		})

		err = cli.Start(ctx)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("leader election test", func(t *testing.T) {
		opts := DefaultOptions()
		opts.LeaderElection = true
		opts.LeaderElectionID = "symops"
		opts.LeaderElectionNamespace = "default"

		cli1, err := NewMingleClient(DefaultClusterCfgInfo(""), opts)
		if err != nil {
			t.Error(err)
			return
		}
		ctx1, cancel1 := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancel1()

		t.Log("start client1")
		go cli1.Start(ctx1)

		cli2, err := NewMingleClient(DefaultClusterCfgInfo(""), opts)
		if err != nil {
			t.Error(err)
			return
		}
		ctx2, cancel2 := context.WithTimeout(context.TODO(), time.Minute*1)
		defer cancel2()

		t.Log("start client2")
		cli2.Start(ctx2)
	})
}

func TestNewMingleClient(t *testing.T) {
	cli, err := NewMingleClient(DefaultClusterCfgInfo(""), DefaultOptions())
	if err != nil {
		t.Error(err)
		return
	}
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go func() {
		err = cli.Start(ctx)
	}()

	if !cli.IsConnected() {
		// maybe run without Kubernetes cluster, should return
		return
	}
}
