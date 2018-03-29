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
	mapper           map[string]*dubbo.Provider
	addrConverter    converter.AddrConverterInterface
	localRegistry    registry.Interface
	remoteRegistry   registry.Interface
	desiredProviders sets.String
	currentProviders sets.String
}

func NewProviderManager(addrConverter converter.AddrConverterInterface, localRegistry registry.Interface, remoteRegistry registry.Interface) *ProviderManager {
	return &ProviderManager{
		mapper:           make(map[string]*dubbo.Provider),
		addrConverter:    addrConverter,
		localRegistry:    localRegistry,
		remoteRegistry:   remoteRegistry,
		desiredProviders: sets.NewString(),
		currentProviders: sets.NewString(),
	}
}

func (m *ProviderManager) Parse(url string) (string, error) {
	provider := dubbo.NewProvider()
	err := provider.Parse(url)
	if err != nil {
		glog.Errorf("parse provider error, err: %v", err)
		return "", err
	}

	addr, err := m.addrConverter.ConvertAddr(provider.Addr)
	if err != nil {
		glog.Errorf("get tlb addr error, err: %v", err)
		return "", err
	}
	provider.Addr = addr

	m.mapper[provider.Key()] = provider

	return provider.Key(), nil
}

func (m *ProviderManager) register(key string) error {
	provider, ok := m.mapper[key]
	if !ok {
		glog.Errorf("provider is not exists, %s", key)
		return fmt.Errorf("provider is not exists")
	}
	return m.remoteRegistry.Register(provider)
}

func (m *ProviderManager) unRegister(key string) {
	provider, ok := m.mapper[key]
	if !ok {
		glog.Errorf("provider is not exists, %s", key)
		return
	}
	m.remoteRegistry.UnRegister(provider)
	delete(m.mapper, key)
	return
}

func (m *ProviderManager) Refresh() {
	urls, err := m.localRegistry.ListProviders()
	if err != nil {
		glog.Errorf("list providers error, %v", err)
		return
	}

	for _, url := range urls {
		provider, err := m.Parse(url)
		if err != nil {
			glog.Warningf("parse provider url error, %v", err)
			continue
		}
		m.desiredProviders.Insert(provider)
	}

	created := m.desiredProviders.Difference(m.currentProviders)
	deleted := m.currentProviders.Difference(m.desiredProviders)

	m.currentProviders = m.desiredProviders
	m.desiredProviders = sets.NewString()

	for provider := range created {
		err := m.register(provider)
		if err != nil {
			glog.Warningf("register provider error, %v", err)
			m.currentProviders.Delete(provider)
			continue
		}
	}
	for provider := range deleted {
		m.unRegister(provider)
	}
}

func (m *ProviderManager) Run(stopCh <-chan struct{}) {
	go m.addrConverter.Run(stopCh)
	go wait.Until(m.Refresh, 10*time.Second, stopCh)
}
