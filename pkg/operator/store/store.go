// Copyright 2021 FabEdge Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"sync"

	apis "github.com/fabedge/fabedge/pkg/apis/v1alpha1"
	"github.com/fabedge/fabedge/pkg/operator/types"
	"github.com/jjeffery/stringset"
)

type Interface interface {
	SaveEndpoint(ep apis.Endpoint)
	SaveEndpointAsLocal(ep apis.Endpoint)
	GetEndpoint(name string) (apis.Endpoint, bool)
	GetEndpoints(names ...string) []apis.Endpoint
	GetAllEndpointNames() stringset.Set
	GetLocalEndpointNames() stringset.Set
	DeleteEndpoint(name string)

	SaveCommunity(ep types.Community)
	GetCommunity(name string) (types.Community, bool)
	GetCommunitiesByEndpoint(name string) []types.Community
	DeleteCommunity(name string)
}

var _ Interface = &store{}

type store struct {
	localNameSet          stringset.Set
	endpoints             map[string]apis.Endpoint
	communities           map[string]types.Community
	endpointToCommunities map[string]stringset.Set

	mux sync.RWMutex
}

func NewStore() Interface {
	return &store{
		localNameSet:          stringset.New(),
		endpoints:             make(map[string]apis.Endpoint),
		communities:           make(map[string]types.Community),
		endpointToCommunities: make(map[string]stringset.Set),
	}
}

func (s *store) SaveEndpoint(ep apis.Endpoint) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.endpoints[ep.Name] = ep
}

func (s *store) SaveEndpointAsLocal(ep apis.Endpoint) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.endpoints[ep.Name] = ep
	s.localNameSet.Add(ep.Name)
}

func (s *store) GetEndpoint(name string) (apis.Endpoint, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	ep, ok := s.endpoints[name]
	return ep, ok
}

func (s *store) GetEndpoints(names ...string) []apis.Endpoint {
	s.mux.Lock()
	defer s.mux.Unlock()

	endpoints := make([]apis.Endpoint, 0, len(names))
	for _, name := range names {
		ep, ok := s.endpoints[name]
		if !ok {
			continue
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

func (s *store) GetAllEndpointNames() stringset.Set {
	s.mux.Lock()
	defer s.mux.Unlock()

	names := make(stringset.Set, len(s.endpoints))
	for name := range s.endpoints {
		names.Add(name)
	}

	return names
}

func (s *store) GetLocalEndpointNames() stringset.Set {
	s.mux.RLock()
	defer s.mux.RUnlock()

	nameSet := stringset.New()
	for name := range s.localNameSet {
		nameSet.Add(name)
	}

	return nameSet
}

func (s *store) DeleteEndpoint(name string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.endpoints, name)
	s.localNameSet.Remove(name)
}

func (s *store) SaveCommunity(c types.Community) {
	s.mux.Lock()
	defer s.mux.Unlock()

	oldCommunity := s.communities[c.Name]
	if oldCommunity.Members.Equal(c.Members) {
		return
	}

	s.communities[c.Name] = c

	// add new member to communities index
	for member := range c.Members {
		cs := s.endpointToCommunities[member]
		cs.Add(c.Name)

		s.endpointToCommunities[member] = cs
	}

	// remove old member to communities index
	for member := range oldCommunity.Members {
		if c.Members.Contains(member) {
			continue
		}

		cs := s.endpointToCommunities[member]
		cs.Remove(c.Name)
		if len(cs) == 0 {
			delete(s.endpointToCommunities, member)
		}
	}
}

func (s *store) GetCommunity(name string) (types.Community, bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	c, ok := s.communities[name]
	return c, ok
}

func (s *store) GetCommunitiesByEndpoint(name string) []types.Community {
	s.mux.Lock()
	defer s.mux.Unlock()

	var communities []types.Community

	cs, ok := s.endpointToCommunities[name]
	if !ok {
		return communities
	}

	for communityName := range cs {
		cmm, ok := s.communities[communityName]
		if ok {
			communities = append(communities, cmm)
		}
	}

	return communities
}

func (s *store) DeleteCommunity(name string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// remove this community from endpointToCommunity
	cmm := s.communities[name]
	for member := range cmm.Members {
		cs := s.endpointToCommunities[member]
		cs.Remove(name)
		if len(cs) == 0 {
			delete(s.endpointToCommunities, member)
		}
	}

	delete(s.communities, name)
}
