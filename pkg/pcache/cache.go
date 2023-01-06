package pcache

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type Cache interface {
	Get(ObjectKindKey) any
	Add(ObjectKindKey, any)
	// TODO timer function
}

func NewCache() Cache {
	return &cache{
		c: map[ObjectKindKey]*cacheData{},
	}
}

type ObjectKindKey struct {
	gvk schema.GroupVersionKind
	nsn types.NamespacedName
}

type cache struct {
	m sync.RWMutex
	c map[ObjectKindKey]*cacheData
}

type cacheData struct {
	data any
	// timer per entry ??
}

func (r *cache) Get(key ObjectKindKey) any {
	r.m.RLock()
	defer r.m.RUnlock()
	if cd, ok := r.c[key]; ok {
		return cd.data
	}
	return nil
}

func (r *cache) Add(key ObjectKindKey, d any) {
	r.m.Lock()
	defer r.m.Unlock()
	r.c[key] = &cacheData{}
}
