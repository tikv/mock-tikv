// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/log"
	"github.com/tikv/mock-tikv/server/api"
)

var tsMu = struct {
	sync.Mutex
	physicalTS int64
	logicalTS  int64
}{}

type idAllocator interface {
	allocID() uint64
}

// Server is the mock tikv server.
type Server struct {
	sync.RWMutex
	// Configs and initial fields.
	cfg *Config

	address string
	ctx     context.Context
	cancel  context.CancelFunc

	idAlloc  uint64
	clusters map[uint64]*clusterInstance

	tso       atomic.Value
	tsoTicker *time.Ticker
}

// CreateServer creates the mock tikv server with given configuration.
func CreateServer(cfg *Config) (s *Server, err error) {
	log.S().Infof("Mock TiKV config - %v", cfg)
	rand.Seed(time.Now().UnixNano())
	var listener net.Listener
	if listener, err = net.Listen("tcp", cfg.ClientEndpoint); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	s = &Server{
		cfg:      cfg,
		address:  strings.Split(cfg.ClientEndpoint, ":")[0],
		clusters: make(map[uint64]*clusterInstance),
		ctx:      ctx,
		cancel:   cancel,
	}

	go http.Serve(listener, api.NewHandler(s))

	return s, nil
}

func (s *Server) allocID() uint64 {
	return atomic.AddUint64(&s.idAlloc, 1)
}

func (s *Server) getTS(count uint32) (physical, logical int64) {
	tsMu.Lock()
	defer tsMu.Unlock()

	ts := time.Now().UnixNano() / int64(time.Millisecond)
	if tsMu.physicalTS >= ts {
		tsMu.logicalTS++
	} else {
		tsMu.physicalTS = ts
		tsMu.logicalTS = 0
	}
	return tsMu.physicalTS, tsMu.logicalTS
}

func (s *Server) doGetCluster(id uint64) *clusterInstance {
	s.RLock()
	defer s.RUnlock()
	if cluster, ok := s.clusters[id]; ok {
		return cluster
	}
	return nil
}

func (s *Server) doGetClusters() []*clusterInstance {
	s.RLock()
	defer s.RUnlock()
	result := make([]*clusterInstance, 0, len(s.clusters))
	for _, instance := range s.clusters {
		result = append(result, instance)
	}
	return result
}

func (s *Server) doDeleteCluster(id uint64) {
	s.Lock()
	defer s.Unlock()
	cluster, ok := s.clusters[id]
	if !ok {
		return
	}
	cluster.stop()
	delete(s.clusters, id)
}

func (s *Server) doCreateCluster(regions []*regionInstance, stores []*storeInstance) (*clusterInstance, error) {
	if len(regions) == 0 {
		regions = []*regionInstance{
			newRegionInstance(0, emptyKey, emptyKey),
		}
	}
	if len(stores) == 0 {
		stores = []*storeInstance{
			newStoreInstance(defaultTiKVVersion),
		}
	}
	if err := validateRegion(regions); err != nil {
		return nil, err
	}
	clusterID := s.allocID()
	regionByID := make(map[uint64]*regionInstance)
	for _, region := range regions {
		region.Id = s.allocID()
		regionByID[region.Id] = region
	}
	storeByID := make(map[uint64]*storeInstance)
	for _, store := range stores {
		store.Id = s.allocID()
		storeByID[store.Id] = store
	}
	instance := &clusterInstance{
		server:       s,
		clusterID:    clusterID,
		maxPeerCount: 3,
		regions:      regions,
		regionByID:   regionByID,
		stores:       stores,
		storeByID:    storeByID,
	}
	instance.member = newMemberInstance(s, instance)
	if err := instance.start(s.address); err != nil {
		return nil, err
	}
	s.clusters[instance.clusterID] = instance
	return instance, nil
}
