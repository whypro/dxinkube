package registry

import "github.com/whypro/dxinkube/pkg/dubbo"

type Interface interface {
	Register(provider *dubbo.Provider) error
	UnRegister(provider *dubbo.Provider) error
	ListProviders() ([]string, error)
}
