package client

import (
	"sync"

	"github.com/symcn/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type proxyMultiClient struct {
	ccm    api.ClusterConfigurationManager
	scheme *runtime.Scheme
	cache  map[string]api.MingleProxyClient
	sync.Mutex
}

func NewMingleProxyClient(ccm api.ClusterConfigurationManager, scheme *runtime.Scheme) api.MultiProxyClient {
	return &proxyMultiClient{
		ccm:    ccm,
		scheme: scheme,
		cache:  make(map[string]api.MingleProxyClient),
	}
}

func (pm *proxyMultiClient) GetAll() []api.MingleProxyClient {
	clsList, err := pm.ccm.GetAll()
	if err != nil {
		klog.Warningf("Not found proxy client config")
		return nil
	}

	cliList := make([]api.MingleProxyClient, 0, len(clsList))
	for _, cls := range clsList {

		cli, ok := pm.GetProxyClientFromCache(cls.GetName())
		if !ok {
			cli, err = NewProxyGatewayMingleClient(cls, pm.scheme)
			if err != nil {
				klog.Errorf("Build proxy client error: %+v, ignore.", err.Error())
				continue
			}
			pm.PutProxyClientToCache(cls.GetName(), cli)
		}
		cliList = append(cliList, cli)
	}

	return cliList
}

func (pm *proxyMultiClient) GetProxyClientFromCache(clsName string) (api.MingleProxyClient, bool) {
	pm.Lock()
	defer pm.Unlock()

	cli, ok := pm.cache[clsName]
	return cli, ok
}

func (pm *proxyMultiClient) PutProxyClientToCache(clsName string, cli api.MingleProxyClient) {
	pm.Lock()
	defer pm.Unlock()

	if len(pm.cache) == 0 {
		pm.cache = make(map[string]api.MingleProxyClient)
	}
	// // TODO: close old client
	// oldCli, ok := pm.cache[clsName]
	// if ok {
	// }
	pm.cache[clsName] = cli
}
