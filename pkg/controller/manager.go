package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/whypro/dxinkube/pkg/converter"
	"github.com/whypro/dxinkube/pkg/dubbo"
	"github.com/whypro/dxinkube/pkg/registry"
)

type ProviderManager struct {
	addrConverter         converter.AddrConverterInterface
	localRegistry         registry.Interface
	remoteRegistry        registry.Interface
	localProvidersMapper  map[string]*dubbo.Provider
	remoteProvidersMapper map[string]*dubbo.Provider
	currentProviders      sets.String
	desiredProviders      sets.String
}

func NewProviderManager(addrConverter converter.AddrConverterInterface, localRegistry registry.Interface, remoteRegistry registry.Interface) *ProviderManager {
	return &ProviderManager{
		addrConverter:         addrConverter,
		localRegistry:         localRegistry,
		remoteRegistry:        remoteRegistry,
		localProvidersMapper:  make(map[string]*dubbo.Provider),
		remoteProvidersMapper: make(map[string]*dubbo.Provider),
		desiredProviders:      sets.NewString(),
		currentProviders:      sets.NewString(),
	}
}

func (m *ProviderManager) Parse(url string, isConvertAddr bool) (*dubbo.Provider, error) {
	provider := dubbo.NewProvider()
	err := provider.Parse(url)
	if err != nil {
		glog.Errorf("parse provider error, err: %v", err)
		return nil, err
	}

	if isConvertAddr {
		addr, err := m.addrConverter.ConvertAddr(provider.Addr)
		if err != nil {
			glog.Errorf("get tlb addr error, err: %v", err)
			return nil, err
		}
		provider.Addr = addr
	}

	return provider, nil
}

func (m *ProviderManager) register(key string) error {
	provider, ok := m.localProvidersMapper[key]
	if !ok {
		glog.Errorf("provider is not exists, %s", key)
		return fmt.Errorf("provider is not exists")
	}
	provider.SetTimestamp()
	return m.remoteRegistry.Register(provider)
}

func (m *ProviderManager) unRegister(key string) error {
	provider, ok := m.remoteProvidersMapper[key]
	if !ok {
		glog.Errorf("provider is not exists, %s", key)
		return fmt.Errorf("provider is not exists")
	}
	return m.remoteRegistry.UnRegister(provider)
}

func (m *ProviderManager) listProviders(r registry.Interface, isConvertAddr bool) (sets.String, map[string]*dubbo.Provider) {
	urls, err := r.ListProviders()
	if err != nil {
		glog.Errorf("list providers error, %v", err)
		return nil, nil
	}

	set := sets.NewString()
	mapper := make(map[string]*dubbo.Provider)
	for _, url := range urls {
		provider, err := m.Parse(url, isConvertAddr)
		if err != nil {
			glog.Warningf("parse provider url error, %v", err)
			continue
		}
		set.Insert(provider.Key())
		mapper[provider.Key()] = provider
	}
	return set, mapper
}

func (m *ProviderManager) Refresh() {
	m.desiredProviders, m.localProvidersMapper = m.listProviders(m.localRegistry, true)
	m.currentProviders, m.remoteProvidersMapper = m.listProviders(m.remoteRegistry, false)

	created := m.desiredProviders.Difference(m.currentProviders)
	deleted := m.currentProviders.Difference(m.desiredProviders)

	for providerKey := range created {
		err := m.register(providerKey)
		if err != nil {
			glog.Warningf("register provider error, %v", err)
			continue
		}
	}

	for providerKey := range deleted {
		m.unRegister(providerKey)
	}
}

func (m *ProviderManager) Run(stopCh <-chan struct{}) {
	go m.addrConverter.Run(stopCh)
	go wait.Until(m.Refresh, 10*time.Second, stopCh)
}
