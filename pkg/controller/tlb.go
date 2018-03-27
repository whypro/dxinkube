package controller

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// podAddr -> tlbAddr
type TLBMapper map[string]string

type TLBController struct {
	kubeClient *kubernetes.Clientset

	tlbMapper TLBMapper
	lock      sync.RWMutex

	endpointsLister   listersv1.EndpointsLister
	serviceLister     listersv1.ServiceLister
	endpointsInformer informersv1.EndpointsInformer
	serviceInformer   informersv1.ServiceInformer
}

func NewTLBController(config *Config) (*TLBController, error) {

	kubeClient, err := kubernetes.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes client")
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, config.ResyncPeriod)
	endpointsLister := informerFactory.Core().V1().Endpoints().Lister()
	serviceLister := informerFactory.Core().V1().Services().Lister()
	endpointsInformer := informerFactory.Core().V1().Endpoints()
	serviceInformer := informerFactory.Core().V1().Services()

	tlbController := &TLBController{
		endpointsLister:   endpointsLister,
		serviceLister:     serviceLister,
		kubeClient:        kubeClient,
		endpointsInformer: endpointsInformer,
		serviceInformer:   serviceInformer,
	}

	endpointsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    tlbController.onEndpointsAdd,
		UpdateFunc: tlbController.onEndpointsUpdate,
		DeleteFunc: tlbController.onEndpointsDelete,
	})
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    nil,
		UpdateFunc: tlbController.onServiceUpdate,
		DeleteFunc: nil,
	})

	return tlbController, nil
}

func (c *TLBController) Run(stopCh <-chan struct{}) {
	go c.endpointsInformer.Informer().Run(stopCh)
	go c.serviceInformer.Informer().Run(stopCh)
}

func (c *TLBController) RefreshTLBMapper() {
	// list tlb services
	selector := labels.NewSelector()
	r, err := labels.NewRequirement(tlbLabelName, selection.Exists, make([]string, 0))
	if err != nil {
		glog.Errorf("create requirement error, err: %v", err)
		return
	}
	selector.Add(*r)
	services, err := c.serviceLister.Services("").List(selector)
	if err != nil {
		glog.Errorf("list services error, err: %v", err)
		return
	}

	// get tlb address
	for _, svc := range services {
		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			glog.Warningf("tlb is not initialized yet, service: %s", svc.Name)
			continue
		}
		// assert one service just has one ingress ip
		var tlbAddr string
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			for _, port := range svc.Spec.Ports {
				tlbAddr = fmt.Sprintf("%s:%d", ingress.IP, port.Port)
			}
		}
		if tlbAddr == "" {
			glog.Warningf("failed to get ips or ports, service: %s", svc.Name)
			continue
		}

		// get endpoints of the service
		ep, err := c.endpointsLister.Endpoints("").Get(svc.Name)
		if err != nil {
			glog.Errorf("get endpoints of service: %s error, err: %v", svc.Name, err)
			continue
		}
		// get pod addresses set
		podAddrs := make([]string, 0)
		for _, subset := range ep.Subsets {
			for _, ip := range subset.Addresses {
				for _, port := range subset.Ports {
					podAddr := fmt.Sprintf("%s:%d", ip.IP, port.Port)
					podAddrs = append(podAddrs, podAddr)
				}
			}
		}
	}

	return
}

func (m *TLBMapper) Update(podAddrs sets.String, tlbAddr string) {
	if podAddrs.Len() == 0 {
		return
	}
	glog.Infof("update tlb mapper")
	glog.Info(m)
}

func (m *TLBMapper) Delete(podAddrs sets.String) {
	if podAddrs.Len() == 0 {
		return
	}
	glog.Infof("delete tlb mapper")
	glog.Info(m)
}

func (m *TLBMapper) String() string {
	return fmt.Sprintf("%+v", m)
}

func getPodAddrsFromEndpoints(ep *v1.Endpoints) sets.String {
	podAddrs := sets.NewString()
	for _, subset := range ep.Subsets {
		for _, ip := range subset.Addresses {
			for _, port := range subset.Ports {
				podAddr := fmt.Sprintf("%s:%d", ip, port)
				podAddrs.Insert(podAddr)
			}
		}
	}
	return podAddrs
}

func diffAddrs(oldAddrs sets.String, newAddrs sets.String) (sets.String, sets.String) {
	deletedAddrs := oldAddrs.Difference(newAddrs)
	createdAddrs := newAddrs.Difference(oldAddrs)
	return deletedAddrs, createdAddrs
}

func (c *TLBController) onEndpointsAdd(obj interface{}) {
	ep, ok := obj.(*v1.Endpoints)
	if !ok {
		glog.Errorf("invalid obj type: %T", obj)
		return
	}
	if !isTLBEndpoints(ep) {
		glog.V(4).Infof("skip endpoints, ns: %s, name: %s", ep.Namespace, ep.Name)
		return
	}
	addrs := getPodAddrsFromEndpoints(ep)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.tlbMapper.Update(addrs, "")
}

func isTLBEndpoints(ep *v1.Endpoints) bool {
	for k := range ep.Labels {
		if tlbLabelName == k {
			return true
		}
	}
	return false
}

func isTLBService(svc *v1.Service) bool {
	for k := range svc.Labels {
		if tlbLabelName == k {
			return true
		}
	}
	return false
}

func (c *TLBController) onEndpointsUpdate(oldObj, newObj interface{}) {
	oldEp, ok := oldObj.(*v1.Endpoints)
	if !ok {
		glog.Errorf("invalid obj type: %T", oldObj)
		return
	}
	if !isTLBEndpoints(oldEp) {
		glog.V(4).Infof("skip endpoints, ns: %s, name: %s", oldEp.Namespace, oldEp.Name)
		return
	}
	newEp, ok := newObj.(*v1.Endpoints)
	if !ok {
		glog.Errorf("invalid obj type: %T", newObj)
		return
	}
	deletedAddrs, createdAddrs := diffAddrs(getPodAddrsFromEndpoints(oldEp), getPodAddrsFromEndpoints(newEp))
	c.lock.Lock()
	defer c.lock.Unlock()
	c.tlbMapper.Delete(deletedAddrs)
	c.tlbMapper.Update(createdAddrs, "")
}

func (c *TLBController) onEndpointsDelete(obj interface{}) {
	ep, ok := obj.(*v1.Endpoints)
	if !ok {
		glog.Errorf("invalid obj type: %T", obj)
		return
	}
	if !isTLBEndpoints(ep) {
		glog.V(4).Infof("skip endpoints, ns: %s, name: %s", ep.Namespace, ep.Name)
		return
	}
	addrs := getPodAddrsFromEndpoints(ep)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.tlbMapper.Delete(addrs)
}

func getTLBAddrFromService(svc *v1.Service) string {
	// assert one service just has one ingress ip
	var tlbAddr string
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		for _, port := range svc.Spec.Ports {
			tlbAddr = fmt.Sprintf("%s:%d", ingress.IP, port.Port)
			// break
		}
	}
	return tlbAddr
}

func (c *TLBController) onServiceUpdate(oldObj, newObj interface{}) {
	oldSvc, ok := oldObj.(*v1.Service)
	if !ok {
		glog.Errorf("invalid obj type: %T", oldObj)
		return
	}
	if !isTLBService(oldSvc) {
		glog.V(4).Infof("skip service, ns: %s, name: %s", oldSvc.Namespace, oldSvc.Name)
		return
	}
	newSvc, ok := newObj.(*v1.Service)
	if !ok {
		glog.Errorf("invalid obj type: %T", newObj)
		return
	}

	tlbAddr := getTLBAddrFromService(newSvc)
	if tlbAddr != "" && getTLBAddrFromService(oldSvc) == "" {

		// get endpoints of the service
		ep, err := c.endpointsLister.Endpoints("").Get(newSvc.Name)
		if err != nil {
			glog.Errorf("get endpoints of service: %s error, err: %v", newSvc.Name, err)
			return
		}
		podAddrs := getPodAddrsFromEndpoints(ep)
		c.lock.Lock()
		defer c.lock.Lock()
		c.tlbMapper.Update(podAddrs, tlbAddr)
	}
	glog.Info(c.tlbMapper)
	return
}

/*

func (c *ZKController) GetTLBAddr(podAddr string) (string, error) {
	c.lock.RLock()
	defer c.lock.Unlock()
	tlbAddr, ok := c.tlbMapper[podAddr]
	if !ok {
		glog.Errorf("podIP %s is not in tlbMapper", podAddr)
		return "", fmt.Errorf("podIP is not in tlbMapper")
	}
	return tlbAddr, nil
}
*/
