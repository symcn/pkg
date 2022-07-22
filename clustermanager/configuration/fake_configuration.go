package configuration

import (
	"github.com/symcn/api"
)

// cfgWithConfigmap clusterconfiguration manager with kubernetes configmap
type FakeConfiguration struct {
	GetAllFunc func() ([]api.ClusterCfgInfo, error)
}

// NewClusterCfgManagerWithCM build cfgWithConfigmap
func NewFakeConfiguration() api.ClusterConfigurationManager {
	return &FakeConfiguration{}
}

// GetAll implements api.ClusterConfigurationManager
func (fc *FakeConfiguration) GetAll() ([]api.ClusterCfgInfo, error) {
	if fc.GetAllFunc == nil {
		return nil, nil
	}
	return fc.GetAllFunc()
}

type FakeClusterCfgInfo struct {
	kubeconfig  string
	configType  api.KubeConfigType
	kubecontext string
	name        string
}

func NewFakeClusterCfgInfo(kubeconfig string, configType api.KubeConfigType, kubecontext string, name string) api.ClusterCfgInfo {
	return &FakeClusterCfgInfo{
		kubeconfig:  kubeconfig,
		configType:  configType,
		kubecontext: kubecontext,
		name:        name,
	}
}

// GetKubeConfig implements api.ClusterCfgInfo
func (fci *FakeClusterCfgInfo) GetKubeConfig() string {
	return fci.kubeconfig
}

// GetKubeConfigType implements api.ClusterCfgInfo
func (fci *FakeClusterCfgInfo) GetKubeConfigType() api.KubeConfigType {
	return fci.configType
}

// GetKubeContext implements api.ClusterCfgInfo
func (fci *FakeClusterCfgInfo) GetKubeContext() string {
	return fci.kubecontext
}

// GetName implements api.ClusterCfgInfo
func (fci *FakeClusterCfgInfo) GetName() string {
	return fci.name
}
