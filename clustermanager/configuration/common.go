package configuration

import "github.com/symcn/api"

type clusterCfgInfo struct {
	name           string
	kubeConfigType api.KubeConfigType
	kubeConfig     string
	kubeContext    string
}

// BuildClusterCfgInfo build api.ClusterCfgInfo
func BuildClusterCfgInfo(name string, kubeConfigType api.KubeConfigType, kubeConfig string, kubeContext string) api.ClusterCfgInfo {
	return &clusterCfgInfo{
		name:           name,
		kubeConfigType: kubeConfigType,
		kubeConfig:     kubeConfig,
		kubeContext:    kubeContext,
	}
}

func (c *clusterCfgInfo) GetName() string {
	return c.name
}

func (c *clusterCfgInfo) GetKubeConfigType() api.KubeConfigType {
	return c.kubeConfigType
}

func (c *clusterCfgInfo) GetKubeConfig() string {
	return c.kubeConfig
}

func (c *clusterCfgInfo) GetKubeContext() string {
	return c.kubeContext
}

// BuildDefaultClusterCfgInfo BuildDefaultClusterCfgInfo with default Kubernetes configuration
// use default ~/.kube/config or Kubernetes cluster internal config
func BuildDefaultClusterCfgInfo(name string) api.ClusterCfgInfo {
	return BuildClusterCfgInfo(name, api.KubeConfigTypeFile, "", "")
}
