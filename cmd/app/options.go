package app

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/whypro/dxinkube/pkg/controller"
	"github.com/whypro/dxinkube/pkg/converter"
	"github.com/whypro/dxinkube/pkg/registry"
)

const (
	defaultServerAddr = "0.0.0.0"
	defaultServerPort = 5000
)

const (
	dubboRootPath             = "/dubbo"
	dubboProviderCategory     = "providers"
	dubboConfiguratorCategory = "configurators"
	zkConnectionTimeout       = 10 * time.Second
	resyncPeriod              = 5 * time.Minute
	tlbLabelName              = "ke-tlb/owner"
)

type ZKControllerOptions struct {
	ServerAddr      string `json:"addr"`
	ServerPort      int32  `json:"port"`
	KubeConfigPath  string `json:"kubeconfig"`
	GlogV           int32  `json:"glog_v"`
	GlogLogtostderr bool   `json:"glog_logtostderr"`
	Version         bool   `json:"version"`

	LocalZKAddrs  []string `json:"local_zk_addrs"`
	RemoteZKAddrs []string `json:"remote_zk_addrs"`

	Namespace string `json:"namespace"`
}

func NewZKControllerOptions() *ZKControllerOptions {
	return &ZKControllerOptions{
		ServerAddr:      defaultServerAddr,
		ServerPort:      defaultServerPort,
		GlogV:           0,
		GlogLogtostderr: true,
	}
}

func (o *ZKControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.ServerAddr, "addr", "h", o.ServerAddr, "")
	fs.Int32VarP(&o.ServerPort, "port", "p", o.ServerPort, "")
	fs.StringVar(&o.KubeConfigPath, "kubeconfig", o.KubeConfigPath, "")

	fs.Int32Var(&o.GlogV, "glog-v", o.GlogV, "")
	fs.BoolVar(&o.GlogLogtostderr, "glog-logtostderr", o.GlogLogtostderr, "")
	fs.BoolVarP(&o.Version, "version", "v", o.Version, "show version")

	fs.StringSliceVar(&o.LocalZKAddrs, "local-zk-addrs", o.LocalZKAddrs, "")
	fs.StringSliceVar(&o.RemoteZKAddrs, "remote-zk-addrs", o.RemoteZKAddrs, "")

	fs.StringVar(&o.Namespace, "namespace", o.Namespace, "")
}

func createZKControllerConfig(o *ZKControllerOptions) *controller.Config {
	var err error

	var kubeClientConfig *rest.Config
	if o.KubeConfigPath != "" {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// if you want to change the loading rules (which files in which order), you can do so here
		loadingRules.ExplicitPath = o.KubeConfigPath
		configOverrides := &clientcmd.ConfigOverrides{}
		// if you want to change override values or bind them to flags, there are methods to help you
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		kubeClientConfig, err = kubeConfig.ClientConfig()
	} else {
		kubeClientConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Fatalf("failed to get kubernetes cluster config: %v", err)
	}

	return &controller.Config{
		TLBConfig: &converter.TLBControllerConfig{
			KubeConfig:   kubeClientConfig,
			TLBLabelName: tlbLabelName,
			ResyncPeriod: resyncPeriod,
			Namespace:    o.Namespace,
		},
		LocalZKConfig: &registry.ZookeeperConfig{
			ServerAddrs:               o.LocalZKAddrs,
			DubboRootPath:             dubboRootPath,
			DubboProviderCategory:     dubboProviderCategory,
			DubboConfiguratorCategory: dubboConfiguratorCategory,
			ConnectionTimeout:         zkConnectionTimeout,
		},
		RemoteZKConfig: &registry.ZookeeperConfig{
			ServerAddrs:               o.RemoteZKAddrs,
			DubboRootPath:             dubboRootPath,
			DubboProviderCategory:     dubboProviderCategory,
			DubboConfiguratorCategory: dubboConfiguratorCategory,
			ConnectionTimeout:         zkConnectionTimeout,
		},
	}
}

func GetGlogCommandLine(fs *pflag.FlagSet) ([]string, error) {
	logtostderr, err := fs.GetBool("glog-logtostderr")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get glog logtostderr failed, err: %v", err)
		return nil, err
	}

	v, err := fs.GetInt32("glog-v")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get glog v failed, err: %v", err)
		return nil, err
	}

	commandLine := []string{
		// -v should be the first arg
		"-v", strconv.FormatInt(int64(v), 10),
		"-logtostderr", strconv.FormatBool(logtostderr),
	}

	return commandLine, nil
}
