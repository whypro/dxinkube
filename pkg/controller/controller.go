package controller

import (
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"k8s.io/client-go/rest"
)

const (
	tlbLabelName = "ke-tlb/owner"
	zkPath       = "/dubbo"
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
	remoteZKClient *zk.Conn
	tlbController  *TLBController
}

func NewZKController(config *Config) (*ZKController, error) {

	localZKClient, _, err := zk.Connect(config.LocalZKAddrs, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", config.LocalZKAddrs, err)
		return nil, err
	}
	remoteZKClient, _, err := zk.Connect(config.RemoteZKAddrs, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", config.LocalZKAddrs, err)
		return nil, err
	}

	tlbController, err := NewTLBController(config)
	if err != nil {
		glog.Errorf("create tlb controller error, err: %v", err)
		return nil, err
	}

	zkController := &ZKController{
		config:         config,
		localZKClient:  localZKClient,
		remoteZKClient: remoteZKClient,
		tlbController:  tlbController,
	}

	return zkController, nil
}

func (c *ZKController) Run(stopCh <-chan struct{}) {
	c.tlbController.Run(stopCh)
	go c.watchLocal(stopCh)
}

func (c *ZKController) watchLocal(stopCh <-chan struct{}) {
	for {
		glog.Infof("watch %s", zkPath)
		// changes := ChildrenWSubscribe(c.localZKClient, zkPath)

		children, _, event, err := c.localZKClient.ChildrenW("/dubbo")
		if err != nil {
			glog.Errorf("children watch error, err: %v", err)
			continue
		}
		glog.Infof("get children: %v", children)
		select {
		case ev := <-event:
			if ev.Err != nil {
				glog.Errorf("event err, %v", ev.Err)
				continue
			}
			glog.Infof("watched: +%v", ev)
		}

		// c.onLocalChange(changes)
	}
}

func (c *ZKController) onLocalChange(changes chan WatchChange) {
	for change := range changes {
		glog.Infof("STATE UPDATED: %+v\n", change)
	}
	glog.Infof("connection closed")
}
