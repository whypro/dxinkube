package controller

import (
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	tlbLabelName  = "ke-tlb/owner"
	dubboRootPath = "/dubbo"

	dubboProviderCategory = "providers"
)

type Config struct {
	KubeConfig    *rest.Config
	ResyncPeriod  time.Duration
	LocalZKAddrs  []string
	RemoteZKAddrs []string
}

type ZKController struct {
	config         *Config
	localZKClient  *zk.Conn
	// remoteZKClient *zk.Conn
	tlbController  *TLBController

	desiredProviderMapper DubboProviderMapper
	currentProviderMapper DubboProviderMapper
}

func NewZKController(config *Config) (*ZKController, error) {

	localZKClient, _, err := zk.Connect(config.LocalZKAddrs, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", config.LocalZKAddrs, err)
		return nil, err
	}
	/*
	remoteZKClient, _, err := zk.Connect(config.RemoteZKAddrs, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", config.LocalZKAddrs, err)
		return nil, err
	}
	*/

	tlbController, err := NewTLBController(config)
	if err != nil {
		glog.Errorf("create tlb controller error, err: %v", err)
		return nil, err
	}

	zkController := &ZKController{
		config:                config,
		localZKClient:         localZKClient,
		// remoteZKClient:        remoteZKClient,
		tlbController:         tlbController,
		desiredProviderMapper: make(map[string]*DubboProvider),
		currentProviderMapper: make(map[string]*DubboProvider),
	}

	return zkController, nil
}

func (c *ZKController) Run(stopCh <-chan struct{}) {
	go c.tlbController.Run(stopCh)
	// go c.watchLocal(stopCh)
	go wait.Until(c.Sync, 10*time.Second, stopCh)
}
