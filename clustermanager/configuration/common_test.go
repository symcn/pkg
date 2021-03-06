package configuration

import (
	"testing"

	"github.com/symcn/api"
)

func TestBuildClusterCfgInfo(t *testing.T) {
	name := "name"
	kubeConfigType := api.KubeConfigTypeFile
	kubeConfig := "kubeConfig"
	kubeContext := "kubeContext"

	cfg := BuildClusterCfgInfo(name, kubeConfigType, kubeConfig, kubeContext)

	if cfg.GetName() != name {
		t.Errorf("ClusterCfgInfo GetName expect %s but got %s", name, cfg.GetName())
		return
	}
	if cfg.GetKubeConfigType() != kubeConfigType {
		t.Errorf("ClusterCfgInfo GetKubeConfigType expect %s but got %s", kubeConfigType, cfg.GetKubeConfigType())
		return
	}
	if cfg.GetKubeConfig() != kubeConfig {
		t.Errorf("ClusterCfgInfo GetKubeConfig expect %s but got %s", kubeConfig, cfg.GetKubeConfig())
		return
	}
	if cfg.GetKubeContext() != kubeContext {
		t.Errorf("ClusterCfgInfo GetKubeContext expect %s but got %s", kubeContext, cfg.GetKubeContext())
		return
	}
}

func TestBuildDefaultClusterCfgInfo(t *testing.T) {
	BuildDefaultClusterCfgInfo("meta")
}
