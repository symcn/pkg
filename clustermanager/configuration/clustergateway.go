package configuration

import (
	"context"

	clustetgatewayv1aplpha1 "github.com/oam-dev/cluster-gateway/pkg/apis/cluster/v1alpha1"
	"github.com/symcn/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type cfgWithClusterGateway struct {
	dynamicInterface dynamic.Interface
	gvr              schema.GroupVersionResource
	cfg              api.ClusterCfgInfo
	filter           FilterHandler
}

func NewClusterCfgManagerWithGateway(dyanamicInterface dynamic.Interface, cfg api.ClusterCfgInfo) api.ClusterConfigurationManager {
	return &cfgWithClusterGateway{
		dynamicInterface: dyanamicInterface,
		gvr:              (&clustetgatewayv1aplpha1.ClusterGateway{}).GetGroupVersionResource(),
		cfg:              cfg,
	}
}

func NewClusterCfgManagerWithGatewayWithFilter(dyanamicInterface dynamic.Interface, cfg api.ClusterCfgInfo, filter FilterHandler) api.ClusterConfigurationManager {
	return &cfgWithClusterGateway{
		dynamicInterface: dyanamicInterface,
		gvr:              (&clustetgatewayv1aplpha1.ClusterGateway{}).GetGroupVersionResource(),
		cfg:              cfg,
		filter:           filter,
	}
}

func (cg *cfgWithClusterGateway) GetAll() ([]api.ClusterCfgInfo, error) {
	list, err := cg.dynamicInterface.Resource(cg.gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	cfgList := make([]api.ClusterCfgInfo, 0, len(list.Items))
	clusterGateway := &clustetgatewayv1aplpha1.ClusterGateway{}
	for _, item := range list.Items {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), clusterGateway)
		if err == nil {
			cfgList = append(cfgList, BuildClusterCfgInfo(item.GetName(), cg.cfg.GetKubeConfigType(), cg.cfg.GetKubeConfig(), cg.cfg.GetKubeContext()))
		}

	}

	cfgList = filterClusterInfo(cfgList, cg.filter)

	return cfgList, nil
}
