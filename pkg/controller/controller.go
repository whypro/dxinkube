package controller

import (
	"github.com/golang/glog"

	"github.com/whypro/dxinkube/pkg/converter"
	"github.com/whypro/dxinkube/pkg/registry"
)

type Config struct {
	LocalZKConfig  *registry.ZookeeperConfig
	RemoteZKConfig *registry.ZookeeperConfig
	TLBConfig      *converter.TLBControllerConfig
	Namespace      string
}

type ZKController struct {
	config          *Config
	providerManager *ProviderManager
}

func NewZKController(config *Config) (*ZKController, error) {

	tlbController, err := converter.NewTLBController(config.TLBConfig)
	if err != nil {
		glog.Errorf("create tlb controller error, err: %v", err)
		return nil, err
	}

	localRegistry, err := registry.NewZookeeperRegistry(config.LocalZKConfig)
	if err != nil {
		glog.Errorf("create local zk registry error, err: %v", err)
		return nil, err
	}

	remoteRegistry, err := registry.NewZookeeperRegistry(config.RemoteZKConfig)
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
