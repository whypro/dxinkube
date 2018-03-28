package registry

import (
	"fmt"
	neturl "net/url"
	"strings"
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

func (r *ZookeeperRegistry) ensurePath(path string) error {
	nodes := strings.Split(path, "/")
	var currentPath string
	for _, node := range nodes {
		if node == "" {
			continue
		}
		currentPath += "/"
		currentPath += node
		exists, _, err := r.conn.Exists(currentPath)
		if err != nil {
			glog.Errorf("check path %s exists error, %v", currentPath, err)
			return err
		}
		if exists == false {
			_, err := r.conn.Create(currentPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
			if err != nil {
				glog.Errorf("create path %s error, %v", currentPath, err)
				return err
			}
		}
	}
	return nil
}

func (r *ZookeeperRegistry) Register(provider *dubbo.Provider) error {
	conn, _, err := zk.Connect(r.servers, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to server %v error, err: %v", r.servers, err)
		return err
	}
	path := dubboRootPath + "/" + provider.Service + "/" + dubboProviderCategory
	err = r.ensurePath(path)
	if err != nil {
		glog.Errorf("ensure path %s error, %v", path, err)
		return err
	}
	provider.AddTimestamp()
	path += "/"
	path += neturl.QueryEscape(provider.String())
	_, err = conn.Create(path, []byte(provider.Addr), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
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
