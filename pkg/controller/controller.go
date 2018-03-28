package controller

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/rest"

	"github.com/whypro/dxinkube/pkg/converter"
	"github.com/whypro/dxinkube/pkg/registry"
)

type Config struct {
	KubeConfig    *rest.Config
	ResyncPeriod  time.Duration
	LocalZKAddrs  []string
	RemoteZKAddrs []string

	DubboRootPath         string
	DubboProviderCategory string
}

type ZKController struct {
	config          *Config
	providerManager *ProviderManager
}

func NewZKController(config *Config) (*ZKController, error) {

	tlbController, err := converter.NewTLBController(config.KubeConfig, config.ResyncPeriod)
	if err != nil {
		glog.Errorf("create tlb controller error, err: %v", err)
		return nil, err
	}

	localRegistry, err := registry.NewZookeeperRegistry(config.LocalZKAddrs)
	if err != nil {
		glog.Errorf("create local zk registry error, err: %v", err)
		return nil, err
	}

	remoteRegistry, err := registry.NewZookeeperRegistry(config.RemoteZKAddrs)
	if err != nil {
		glog.Errorf("create remote zk registry error, err: %v", err)
		return nil, err
	}

	dubboProviderManager := NewProviderManager(tlbController, localRegistry, remoteRegistry)

	zkController := &ZKController{
		config:          config,
		providerManager: dubboProviderManager,
	}

	return zkController, nil
}

func (c *ZKController) Run(stopCh <-chan struct{}) {
	go c.providerManager.Run(stopCh)
}
