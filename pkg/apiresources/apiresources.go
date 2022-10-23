package apiresources

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yndd/ndd-runtime/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type APIResources interface {
	Refresh()
	HasSynced() bool
	Get(apiVersion, resource string) (result *APIResource)
	GetKind(apiVersion, resource string) (result *APIResource)
	PrintKinds()
	PrintResources()
}

type groupVersionEntry struct {
	resources, subresources, kinds map[string]*APIResource
}

type apiresources struct {
	m             sync.RWMutex
	groupVersions map[string]groupVersionEntry

	discoveryClient discovery.DiscoveryInterface
	Logger          logging.Logger
}

type Options struct {
	Logger logging.Logger
}

func New(discoveryClient *discovery.DiscoveryClient, opt *Options) APIResources {
	return &apiresources{
		discoveryClient: discoveryClient,
		Logger:          opt.Logger,
	}
}

func (r *apiresources) Refresh() {
	log := r.Logger
	log.Info("refreshing resource list")
	_, groups, err := r.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		if len(groups) == 0 {
			log.Info("Failed to fetch discovery info", "error", err)
			return
		}
		log.Info("Failed to fetch all resources, continuing with partial discovery info", "error", err)
	}

	// Denormalize resource lists into maps for convenient lookup
	// by either Group-Version-Kind or Group-Version-Resource.
	groupVersions := make(map[string]groupVersionEntry, len(groups))

	for _, group := range groups {
		gv, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			// This shouldn't happen because we get these values from the server.
			panic(fmt.Errorf("received invalid GroupVersion from server: %s", err.Error()))
		}
		fmt.Printf("group: %s, version: %s\n", gv.Group, gv.Version)
		gve := groupVersionEntry{
			resources:    make(map[string]*APIResource, len(group.APIResources)),
			kinds:        make(map[string]*APIResource, len(group.APIResources)),
			subresources: make(map[string]*APIResource, len(group.APIResources)),
		}

		for i := range group.APIResources {
			fmt.Printf("groupVersion: %s &apiResource: %v\n", group.GroupVersion, group.APIResources[i].Name)
			apiResource := &APIResource{
				APIResource: group.APIResources[i],
				APIVersion:  group.GroupVersion,
			}
			// Materialize default values from the list into each entry.
			if apiResource.Group == "" {
				apiResource.Group = gv.Group
			}
			if apiResource.Version == "" {
				apiResource.Version = gv.Version
			}
			gve.resources[apiResource.Name] = apiResource
			// Remember which resources are subresources, and map the kind to the main resource.
			// This is different from what RESTMapper provides because we already know
			// the full GroupVersionKind and just need the resource name.
			if strings.ContainsRune(apiResource.Name, '/') {
				gve.subresources[apiResource.Name] = apiResource
			} else {
				gve.kinds[apiResource.Kind] = apiResource
			}

			// Group all subresources for a resource.
			for apiSubresourceName := range gve.subresources {
				arr := strings.Split(apiSubresourceName, "/")
				apiResourceName := arr[0]
				subresourceKey := arr[1]
				apiResource := gve.resources[apiResourceName]
				if apiResource == nil {
					continue
				}
				if apiResource.subresourceMap == nil {
					apiResource.subresourceMap = make(map[string]bool)
				}
				apiResource.subresourceMap[subresourceKey] = true
			}

			groupVersions[group.GroupVersion] = gve
		}

		// Replace the local cache.
		r.m.Lock()
		r.groupVersions = groupVersions
		r.m.Unlock()
	}
}

func (r *apiresources) HasSynced() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.groupVersions != nil
}

func (r *apiresources) Get(apiVersion, resource string) (result *APIResource) {
	r.m.RLock()
	defer r.m.RUnlock()

	gv, ok := r.groupVersions[apiVersion]
	if !ok {
		return nil
	}
	return gv.resources[resource]
}

func (r *apiresources) GetKind(apiVersion, kind string) (result *APIResource) {
	r.m.RLock()
	defer r.m.RUnlock()

	gv, ok := r.groupVersions[apiVersion]
	if !ok {
		return nil
	}
	return gv.kinds[kind]
}

func (r *apiresources) PrintResources() {
	r.m.RLock()
	defer r.m.RUnlock()
	fmt.Println("##### print resource #########")
	for apiVersion, gve := range r.groupVersions {
		for name, v := range gve.resources {
			fmt.Printf("apiVersion: %s resource: %s, name: %s\n", apiVersion, name, v.Name)
		}
	}
}

func (r *apiresources) PrintKinds() {
	r.m.RLock()
	defer r.m.RUnlock()
	fmt.Println("##### print kinds #########")
	for apiVersion, gve := range r.groupVersions {
		for name, v := range gve.kinds {
			fmt.Printf("apiVersion: %s kind: %s, name: %s\n", apiVersion, name, v.Name)
		}
	}
}
