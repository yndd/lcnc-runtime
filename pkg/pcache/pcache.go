package pcache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/henderiw-k8s-lcnc/discovery/registrator"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/alloc/alloc"
	"github.com/nokia/k8s-ipam/pkg/alloc/allocpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type ProxyCache interface {
	// Discovers the server
	Start(context.Context)
	// Subscribe to the State Cache for the (ipam, etc) GVK
	// Allocation -> lookup in local cache based on (ipam, etc) gvknsn
	// Timer based refresh Config/State Cache
	// NotifyClient controller though generic event (informer) -> all the For
}

type Config struct {
	Registrator   registrator.Registrator
	EventChannels map[schema.GroupVersionKind]chan event.GenericEvent
}

func New(c *Config) ProxyCache {
	l := ctrl.Log.WithName("k8s-registrator")

	return &pcache{
		informer:    NewInformer(c.EventChannels),
		cache:       NewCache(),
		registrator: c.Registrator,
		l:           l,
	}
}

type pcache struct {
	informer Informer
	// this is the ipam GVK namespace, name
	cache Cache
	// registrator finds the ipam
	registrator registrator.Registrator
	svcInfo     *registrator.Service
	allocClient alloc.Client
	//logger
	l logr.Logger
}

func (r *pcache) Allocate(u *unstructured.Unstructured) {
	key := ObjectKindKey{
		gvk: u.GroupVersionKind(),
		nsn: types.NamespacedName{
			Namespace: u.GetNamespace(),
			Name:      u.GetName(),
		},
	}
	d := r.cache.Get(key)
	if d != nil {
		// check if the data is consistent and if there is an allocation
		r.l.Info("cache hit, data is valid or not: TBD")
	}
	// allocate the
	r.l.Info("no cache hit, retreive data from source")
}

func (r *pcache) Start(ctx context.Context) {
	r.l.Info("starting proxy cache")
	ch := r.registrator.Watch(ctx, "ipam", []string{}, registrator.WatchOptions{RetriveServices: true})

	go func() {
	GeneralWait:
		for {
			select {
			case svcInfo := <-ch:
				r.l.Info("service", "info", *svcInfo)

				if r.svcInfo != nil {
					for _, service := range svcInfo.ServiceInstances {
						if service.Address == r.svcInfo.Address &&
							service.Port == r.svcInfo.Port &&
							service.ID == r.svcInfo.ID &&
							service.Name == r.svcInfo.Name {
							r.l.Info("service, no change, keep waiting", "allocClient", r.allocClient.Get())
							resp, err := r.allocClient.Get().Allocation(context.TODO(), &allocpb.Request{
								Meta: &allocpb.Meta{
									Namespace: "default",
									Name:      "dummyAlloc",
								},
								Spec: &allocpb.Spec{
									Attributes: map[string]string{
										ipamv1alpha1.NephioPrefixKindKey:      string(ipamv1alpha1.PrefixKindLoopback),
										ipamv1alpha1.NephioNetworkInstanceKey: "vpc-mgmt2",
									},
									Selector: map[string]string{
										"nephio.org/fabric":  "fabric1",
										"nephio.org/purpose": "mgmt",
									},
								},
							})
							if err != nil {
								r.l.Error(err, "cannot get allocation§")
							} else {
								r.l.Info("service, no change, keep waiting", "resp", resp)
							}

							continue GeneralWait
						}
					}
				}
				if len(svcInfo.ServiceInstances) == 0 {
					r.l.Info("service, no available service -> delete Client")
					// delete client
					if r.allocClient != nil {
						if err := r.allocClient.Delete(); err != nil {
							r.l.Error(err, "cannot delete client")
						}
						continue GeneralWait
					}
					r.allocClient = nil

				} else {
					r.svcInfo = svcInfo.ServiceInstances[0]
					r.l.Info("service, info changed-> delete and create Client", "svcInfo", svcInfo.ServiceInstances[0])

					// delete client
					if r.allocClient != nil {
						if err := r.allocClient.Delete(); err != nil {
							r.l.Error(err, "cannot delete client")
						}
						continue GeneralWait
					}
					// create client
					ac, err := alloc.New(&alloc.Config{
						Address:  fmt.Sprintf("%s:%s", r.svcInfo.Address, strconv.Itoa(r.svcInfo.Port)),
						Insecure: true,
					})
					if err != nil {
						r.l.Error(err, "cannot create client")
						r.allocClient = nil
						continue GeneralWait
					}
					r.allocClient = ac

				}
			case <-ctx.Done():
				// called when the controller gets cancelled
				return
			}
		}
	}()
}
