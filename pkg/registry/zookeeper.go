package registry

import (
	neturl "net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"

	"github.com/whypro/dxinkube/pkg/dubbo"
)

const (
	dubboRootPath             = "/dubbo"
	dubboProviderCategory     = "providers"
	dubboConfiguratorCategory = "configurators"
	defaultConnectionTimeout  = 10 * time.Second
)

type ZookeeperRegistry struct {
	servers []string
	conn    *zk.Conn
}

func NewZookeeperRegistry(servers []string) (*ZookeeperRegistry, error) {
	conn, _, err := zk.Connect(servers, defaultConnectionTimeout)
	if err != nil {
		glog.Errorf("connect to zk error, addrs: %+v, err: %v", servers, err)
		return nil, err
	}

	registry := &ZookeeperRegistry{
		servers: servers,
		conn:    conn,
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

func (r *ZookeeperRegistry) deletePath(path string) error {
	nodes, _, err := r.conn.Children(path)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return nil
	}
	for _, node := range nodes {
		r.deletePath(path + "/" + node)
	}
	return r.conn.Delete(path, 0)
}

func (r *ZookeeperRegistry) getProvidersPath(provider *dubbo.Provider) string {
	return dubboRootPath + "/" + provider.Service + "/" + dubboProviderCategory
}

func (r *ZookeeperRegistry) getConfiguratorsPath(provider *dubbo.Provider) string {
	return dubboRootPath + "/" + provider.Service + "/" + dubboConfiguratorCategory
}

func (r *ZookeeperRegistry) getProviderPath(provider *dubbo.Provider) string {
	return r.getProvidersPath(provider) + "/" + neturl.QueryEscape(provider.String())
}

func (r *ZookeeperRegistry) getServicePath(provider *dubbo.Provider) string {
	return dubboRootPath + "/" + provider.Service
}

func (r *ZookeeperRegistry) Register(provider *dubbo.Provider) error {
	providersPath := r.getProvidersPath(provider)
	err := r.ensurePath(providersPath)
	if err != nil {
		glog.Errorf("ensure path %s error, %v", providersPath, err)
		return err
	}
	configuratorPath := r.getConfiguratorsPath(provider)
	err = r.ensurePath(configuratorPath)
	if err != nil {
		glog.Errorf("ensure path %s error, %v", configuratorPath, err)
		return err
	}
	path := r.getProviderPath(provider)
	_, err = r.conn.Create(path, []byte(provider.Addr), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("create path %s error, err: %v", path, err)
		return err
	}
	return nil
}

func (r *ZookeeperRegistry) checkEmpty(path string) (bool, error) {
	node, _, err := r.conn.Children(path)
	if err != nil {
		return false, err
	}
	if len(node) != 0 {
		return false, nil
	}
	return true, nil
}

func (r *ZookeeperRegistry) UnRegister(provider *dubbo.Provider) error {
	path := r.getProviderPath(provider)
	err := r.conn.Delete(path, 0)
	if err != nil {
		glog.Errorf("delete path %s error, err: %v", path, err)
		return err
	}

	providersPath := r.getProvidersPath(provider)
	isEmpty, err := r.checkEmpty(providersPath)
	if err != nil {
		glog.Warningf("check path empty error, path: %s, err: %v", providersPath, err)
		return nil
	}
	if isEmpty {
		servicePath := r.getServicePath(provider)
		_ = r.deletePath(servicePath)
	}

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
