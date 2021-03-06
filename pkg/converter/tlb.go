package converter

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// podAddr -> tlbAddr
type TLBMapper map[string]string

func (m TLBMapper) Update(podAddrs sets.String, tlbAddr string) {
	if podAddrs.Len() == 0 || tlbAddr == "" {
		return
	}
	glog.V(4).Infof("update tlb mapper, podAddrs: %s, tlbAddr: %s", podAddrs, tlbAddr)
	for podAddr := range podAddrs {
		m[podAddr] = tlbAddr
	}
}

func (m TLBMapper) Delete(podAddrs sets.String) {
	if podAddrs.Len() == 0 {
		return
	}
	glog.V(4).Infof("delete tlb mapper, podAddrs: %s", podAddrs)
	for podAddr := range podAddrs {
		delete(m, podAddr)
	}
}

type TLBControllerConfig struct {
	KubeConfig   *rest.Config
	TLBLabelName string
	ResyncPeriod time.Duration
	Namespace    string
}

type TLBController struct {
	config     *TLBControllerConfig
	kubeClient *kubernetes.Clientset

	tlbMapper TLBMapper
	lock      sync.RWMutex

	endpointsLister   listersv1.EndpointsLister
	serviceLister     listersv1.ServiceLister
	endpointsInformer informersv1.EndpointsInformer
	serviceInformer   informersv1.ServiceInformer
}

func NewTLBController(config *TLBControllerConfig) (*TLBController, error) {

	kubeClient, err := kubernetes.NewForConfig(config.KubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes client")
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, config.ResyncPeriod)

	endpointsInformer := informerFactory.Core().V1().Endpoints()
	serviceInformer := informerFactory.Core().V1().Services()
	endpointsLister := endpointsInformer.Lister()
	serviceLister := serviceInformer.Lister()

	tlbController := &TLBController{
		config:     config,
		kubeClient: kubeClient,

		tlbMapper: make(TLBMapper),

		endpointsLister:   endpointsLister,
		serviceLister:     serviceLister,
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
	go wait.Until(c.RefreshTLBMapper, 10*time.Second, stopCh)
}

func (c *TLBController) RefreshTLBMapper() {
	// list tlb services
	glog.V(4).Infof("list tlb services")
	selector := labels.NewSelector()
	r, err := labels.NewRequirement(c.config.TLBLabelName, selection.Exists, nil)
	if err != nil {
		glog.Errorf("create requirement error, err: %v", err)
		return
	}
	selector.Add(*r)
	services, err := c.serviceLister.Services(c.config.Namespace).List(selector)
	if err != nil {
		glog.Errorf("list services error, err: %v", err)
		return
	}

	// get tlb address
	for _, svc := range services {
		if !c.isTLBService(svc) {
			glog.V(7).Infof("skip non-tlb service, ns: %s, name: %s", svc.Namespace, svc.Name)
			continue
		}
		tlbAddr := getTLBAddrFromService(svc)
		if tlbAddr == "" {
			glog.V(4).Infof("tlb service not initialized yet, ns: %s, name: %s", svc.Namespace, svc.Name)
			continue
		}
		// get endpoints of the service
		ep, err := c.endpointsLister.Endpoints(svc.Namespace).Get(svc.Name)
		if err != nil {
			glog.Warningf("get endpoints of service error, ns: %s, name: %s, err: %v", svc.Namespace, svc.Name, err)
			continue
		}
		glog.V(4).Infof("got valid service, ns: %s, name: %s", svc.Namespace, svc.Name)
		podAddrs := getPodAddrsFromEndpoints(ep)
		c.lock.Lock()
		c.tlbMapper.Update(podAddrs, tlbAddr)
		c.lock.Unlock()
	}

	return
}

func getPodAddrsFromEndpoints(ep *v1.Endpoints) sets.String {
	podAddrs := sets.NewString()
	for _, subset := range ep.Subsets {
		for _, ip := range subset.Addresses {
			for _, port := range subset.Ports {
				podAddr := fmt.Sprintf("%s:%d", ip.IP, port.Port)
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
	if !c.isTLBEndpoints(ep) {
		glog.V(7).Infof("skip endpoints, ns: %s, name: %s", ep.Namespace, ep.Name)
		return
	}
	addrs := getPodAddrsFromEndpoints(ep)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.tlbMapper.Update(addrs, "")
}

func (c *TLBController) isTLBEndpoints(ep *v1.Endpoints) bool {
	if ep.Namespace != c.config.Namespace {
		return false
	}
	for k := range ep.Labels {
		if c.config.TLBLabelName == k {
			return true
		}
	}
	return false
}

func (c *TLBController) isTLBService(svc *v1.Service) bool {
	if svc.Namespace != c.config.Namespace {
		return false
	}
	for k := range svc.Labels {
		if c.config.TLBLabelName == k {
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
	if !c.isTLBEndpoints(oldEp) {
		glog.V(7).Infof("skip endpoints, ns: %s, name: %s", oldEp.Namespace, oldEp.Name)
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
	if !c.isTLBEndpoints(ep) {
		glog.V(7).Infof("skip endpoints, ns: %s, name: %s", ep.Namespace, ep.Name)
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
	if !c.isTLBService(oldSvc) {
		glog.V(7).Infof("skip service, ns: %s, name: %s", oldSvc.Namespace, oldSvc.Name)
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
		ep, err := c.endpointsLister.Endpoints(newSvc.Namespace).Get(newSvc.Name)
		if err != nil {
			glog.Errorf("get endpoints of service error, ns: %s, name: %s, err: %v", newSvc.Namespace, newSvc.Name, err)
			return
		}
		podAddrs := getPodAddrsFromEndpoints(ep)
		c.lock.Lock()
		defer c.lock.Unlock()
		c.tlbMapper.Update(podAddrs, tlbAddr)
	}
	return
}

func (c *TLBController) ConvertAddr(podAddr string) (string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	tlbAddr, ok := c.tlbMapper[podAddr]
	if !ok {
		glog.Errorf("podIP %s is not in tlbMapper", podAddr)
		return "", fmt.Errorf("podIP is not in tlbMapper")
	}
	return tlbAddr, nil
}
