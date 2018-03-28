package controller

import (
	"fmt"
	"strings"
	"time"
	neturl "net/url"

	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
)

type DubboProviderMapper map[string]*DubboProvider

func (m DubboProviderMapper) Update(provider *DubboProvider) {
	glog.Infof("update provider, %s", provider)
}

func (m DubboProviderMapper) Difference(n DubboProviderMapper) []*DubboProvider {
	result := make([]*DubboProvider, 0)
	for k := range m {
		v, ok := n[k]
		if !ok {
			result = append(result, v)
		}
	}
	return result
}

func (m DubboProviderMapper) Clear() {
	for k := range m {
		delete(m, k)
	}
}

type DubboProvider struct {
	scheme  string
	addr    string
	service string
	params  map[string]string
	conn    *zk.Conn
}

func NewDubboProvider() *DubboProvider {
	return &DubboProvider{
		params: make(map[string]string),
	}
}

func (p *DubboProvider) url() string {
	return fmt.Sprintf("%s://%s/%s", p.scheme, p.addr, p.service)
}

func (p *DubboProvider) query() string {
	paramsSlice := make([]string, 0, len(p.params))
	for k, v := range p.params {
		paramsSlice = append(paramsSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(paramsSlice, "&")
}

func (p *DubboProvider) String() string {
	return p.url() + "?" + p.query()
}

func (p *DubboProvider) Key() string {
	return p.url()
}

func (p *DubboProvider) Parse(url string) error {
	unescapedURL, err := neturl.PathUnescape(url)
	if err != nil {
		return err
	}
	// TODO: use RE
	schemeAndOther := strings.Split(unescapedURL, "://")
	p.scheme = schemeAndOther[0]
	urlAndPath := strings.Split(schemeAndOther[1], "/")
	p.addr = urlAndPath[0]
	pathAndParams := strings.Split(urlAndPath[1], "?")
	p.service = pathAndParams[0]
	params := strings.Split(pathAndParams[1], "&")
	for _, param := range params {
		pSlice := strings.Split(param, "=")
		k, v := pSlice[0], pSlice[1]
		if k == "timestamp" {
			continue
		}
		p.params[k] = v
	}
	glog.Info(url)
	glog.Info(p)
	return nil
}

func (p *DubboProvider) SetAddr(addr string) {
	p.addr = addr
}

func (p *DubboProvider) GetAddr() string {
	return p.addr
}

func (p *DubboProvider) Register(server []string) error {
	conn, _, err := zk.Connect(server, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to server %v error, err: %v", server, err)
		return err
	}
	path := dubboRootPath + "/" + p.service + "/" + dubboProviderCategory + "/" + p.String()
	_, err = conn.Create(path, []byte(p.addr), zk.FlagEphemeral, nil)
	if err != nil {
		glog.Errorf("create path %s error, err: %v", path, err)
		return err
	}
	p.conn = conn
	return nil
}

func (p *DubboProvider) UnRegister() {
	if p.conn == nil {
		glog.Warningf("zk conn is nil, provider url: %s", p.url())
		return
	}
	p.conn.Close()
}

func (c *ZKController) Sync() {
	err := c.ListProviders()
	if err != nil {
		glog.Errorf("list providers error: %v", err)
		return
	}
	c.UpdateProviders()
}

func (c *ZKController) ListProviders() error {
	glog.V(4).Infof("list providers")
	rootPath := dubboRootPath
	children, _, err := c.localZKClient.Children(rootPath)
	if err != nil {
		glog.Errorf("get children for path %s error, err: %v", rootPath, err)
		return err
	}

	for _, child := range children {
		providersPath := rootPath + "/" + child + "/" + dubboProviderCategory

		children, _, err := c.localZKClient.Children(providersPath)
		if err != nil {
			glog.Errorf("get children for path %s error, err: %v", providersPath, err)
			return err
		}

		for _, child := range children {
			// providerPath := providersPath + "/" + child
			provider := NewDubboProvider()
			err := provider.Parse(child)
			if err != nil {
				glog.Errorf("parse provider error, err: %v", err)
				continue
			}

			tlbAddr, err := c.tlbController.GetTLBAddr(provider.GetAddr())
			if err != nil {
				glog.Errorf("get tlb addr error, err: %v", err)
				continue
			}

			provider.SetAddr(tlbAddr)
			c.desiredProviderMapper.Update(provider)
		}
	}

	return nil
}

func (c *ZKController) UpdateProviders() {
	created := c.desiredProviderMapper.Difference(c.currentProviderMapper)
	deleted := c.currentProviderMapper.Difference(c.desiredProviderMapper)

	for _, provider := range created {
		provider.Register(c.config.RemoteZKAddrs)
	}
	for _, provider := range deleted {
		provider.UnRegister()
	}

	c.currentProviderMapper = c.desiredProviderMapper
	c.desiredProviderMapper.Clear()
}
