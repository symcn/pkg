package configuration

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/symcn/api"
	"k8s.io/klog/v2"
)

// cfgWithPath clusterconfiguration manager with file path
type cfgWithPath struct {
	dir            string
	suffix         string
	kubeConfigType api.KubeConfigType
}

// NewClusterCfgManagerWithPath build cfgWithPath
func NewClusterCfgManagerWithPath(dir string, suffix string, kubeConfigType api.KubeConfigType) (api.ClusterConfigurationManager, error) {
	s, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("NewClusterCfgManagerWithPath %s is not exist %+v", dir, err)
	}
	if !s.IsDir() {
		return nil, fmt.Errorf("NewClusterCfgManagerWithPath %s is not directory", dir)
	}

	return &cfgWithPath{
		dir:            dir,
		suffix:         suffix,
		kubeConfigType: kubeConfigType,
	}, nil
}

func (cp *cfgWithPath) GetAll() ([]api.ClusterCfgInfo, error) {
	files, err := ioutil.ReadDir(cp.dir)
	if err != nil {
		return nil, fmt.Errorf("get clusterconfiguration with path failed, open %s err %+v", cp.dir, err)
	}

	list := make([]api.ClusterCfgInfo, 0, len(files))
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), cp.suffix) {
			continue
		}

		path := cp.dir + "/" + file.Name()

		switch cp.kubeConfigType {

		case api.KubeConfigTypeFile:
			list = append(list, BuildClusterCfgInfo(file.Name(), cp.kubeConfigType, path, ""))

		case api.KubeConfigTypeRawString:
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("get clusterconfiguration read %s err %+v", path, err)
			}
			list = append(list, BuildClusterCfgInfo(file.Name(), cp.kubeConfigType, string(data), ""))

		default:
			klog.Warningf("Get clusterconfiguration with path not support type %s", cp.kubeConfigType)
		}
	}

	return list, nil
}
