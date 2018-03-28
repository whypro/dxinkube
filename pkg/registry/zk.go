package registry

import (
	"fmt"
	neturl "net/url"
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"

	"github.com/whypro/dxinkube/pkg/dubbo"
)

const (
	dubboRootPath         = "/dubbo"
	dubboProviderCategory = "providers"
)

type ZookeeperRegistry struct {
	servers           []string
	registerQueue     []*dubbo.Provider
	unRegisterQueue   []*dubbo.Provider
	ephemeralConnPool map[string]*zk.Conn
	conn              *zk.Conn
}

func NewZookeeperRegistry(servers []string) (*ZookeeperRegistry, error) {
	conn, _, err := zk.Connect(servers, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", servers, err)
		return nil, err
	}

	registry := &ZookeeperRegistry{
		servers:           servers,
		registerQueue:     make([]*dubbo.Provider, 0),
		unRegisterQueue:   make([]*dubbo.Provider, 0),
		ephemeralConnPool: make(map[string]*zk.Conn),
		conn:              conn,
	}
	return registry, nil
}

func (r *ZookeeperRegistry) Register(provider *dubbo.Provider) error {
	conn, _, err := zk.Connect(r.servers, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to server %v error, err: %v", r.servers, err)
		return err
	}
	path := dubboRootPath + "/" + provider.Service + "/" + dubboProviderCategory + "/" + provider.String()
	path = neturl.PathEscape(path)
	_, err = conn.Create(path, []byte(provider.Addr), zk.FlagEphemeral, nil)
	if err != nil {
		glog.Errorf("create path %s error, err: %v", path, err)
		return err
	}
	r.ephemeralConnPool[provider.Key()] = conn
	return nil
}

func (r *ZookeeperRegistry) UnRegister(provider *dubbo.Provider) error {
	conn, ok := r.ephemeralConnPool[provider.Key()]
	if !ok {
		glog.Errorf("get provider error, key: %s", provider.Key())
		return fmt.Errorf("provider is not exists")
	}
	conn.Close()
	return nil
}

func (r *ZookeeperRegistry) ListProviders() ([]string, error) {
	glog.V(4).Infof("list providers")
	rootPath := dubboRootPath
	children, _, err := r.conn.Children(rootPath)
	if err != nil {
		glog.Errorf("get children for path %s error, err: %v", rootPath, err)
		return nil, err
	}

	providers := make([]string, 0)
	for _, service := range children {
		providersPath := rootPath + "/" + service + "/" + dubboProviderCategory

		children, _, err := r.conn.Children(providersPath)
		if err != nil {
			glog.Errorf("get children for path %s error, err: %v", providersPath, err)
			return nil, err
		}

		for _, child := range children {
			providers = append(providers, child)
		}
	}

	return providers, nil
}
