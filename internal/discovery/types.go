/*
Copyright 2023 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package discovery

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type groupVersionKindPlural struct {
	schema.GroupVersionKind
	Plural string
}

func (g groupVersionKindPlural) String() string {
	return g.Group + "/" + g.Version + ", Kind=" + g.Kind + ", Plural=" + g.Plural
}

type kindPlural struct {
	Kind   string
	Plural string
}

// CRDiscoverer provides a cache of the collected GVKs, along with helper utilities.
type CRDiscoverer struct {
	// m is a mutex to protect the cache.
	m sync.RWMutex
	// Map is a cache of the collected GVKs.
	Map map[string]map[string][]kindPlural
	// ShouldUpdate is a flag that indicates whether the cache was updated.
	WasUpdated bool
	// CRDsAddEvents is the number of times that the CRD informer triggered the "add" event.
	CRDsAddEvents uint
	// CRDsDeleteEvents is the number of times that the CRD informer triggered the "remove" event.
	CRDsDeleteEvents uint
	// CRDsCacheCount is the net amount of CRDs affecting the cache at this point.
	CRDsCacheCount uint
}

// SafeRead executes the given function while holding a read lock.
func (r *CRDiscoverer) SafeRead(f func()) {
	r.m.RLock()
	defer r.m.RUnlock()
	f()
}

// SafeWrite executes the given function while holding a write lock.
func (r *CRDiscoverer) SafeWrite(f func()) {
	r.m.Lock()
	defer r.m.Unlock()
	f()
}

// AppendToMap appends the given GVKs to the cache.
func (r *CRDiscoverer) AppendToMap(gvkps ...groupVersionKindPlural) {
	if r.Map == nil {
		r.Map = map[string]map[string][]kindPlural{}
	}
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			r.Map[gvkp.Group] = map[string][]kindPlural{}
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			r.Map[gvkp.Group][gvkp.Version] = []kindPlural{}
		}
		r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version], kindPlural{Kind: gvkp.Kind, Plural: gvkp.Plural})
	}
}

// RemoveFromMap removes the given GVKs from the cache.
func (r *CRDiscoverer) RemoveFromMap(gvkps ...groupVersionKindPlural) {
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			continue
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			continue
		}
		for i, el := range r.Map[gvkp.Group][gvkp.Version] {
			if el.Kind == gvkp.Kind {
				if len(r.Map[gvkp.Group][gvkp.Version]) == 1 {
					delete(r.Map[gvkp.Group], gvkp.Version)
					if len(r.Map[gvkp.Group]) == 0 {
						delete(r.Map, gvkp.Group)
					}
					break
				}
				r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version][:i], r.Map[gvkp.Group][gvkp.Version][i+1:]...)
				break
			}
		}
	}
}
