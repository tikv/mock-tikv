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
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultTiKVVersion = "3.0.0"

var (
	emptyKey = []byte{}
)

type clusterInstance struct {
	sync.RWMutex
	server       *Server
	clusterID    uint64
	maxPeerCount uint32
	safePoint    uint64

	member     *memberInstance // we only support one leader pd at this time
	regions    []*regionInstance
	regionByID map[uint64]*regionInstance
	stores     []*storeInstance
	storeByID  map[uint64]*storeInstance

	idAlloc uint64

	store *MVCCLevelDB
}

func validateRegion(regions []*regionInstance) error {
	var lastEndKey []byte
	var lastRegion *regionInstance
	for _, region := range regions {
		if lastEndKey == nil {
			if len(region.StartKey) != 0 {
				return errors.New(fmt.Sprintf("Start key of first region '%d' isn't empty", region.Id))
			}
			if region.EndKey == nil {
				region.EndKey = emptyKey
			}
		} else {
			if !bytes.Equal(region.StartKey, lastEndKey) {
				return errors.New(fmt.Sprintf("Start key '%s' of region '%d' isn't the same as end key '%s' of last region '%d'",
					hex.EncodeToString(region.StartKey), region.Id,
					hex.EncodeToString(lastEndKey), lastRegion.Id,
				))
			}
		}
		if len(region.EndKey) > 0 && len(region.StartKey) > 0 && bytes.Compare(region.StartKey, region.EndKey) >= 0 {
			return errors.New(fmt.Sprintf("Start key '%s' of region '%d' isn't the less than end key '%s'",
				hex.EncodeToString(region.StartKey), region.Id,
				hex.EncodeToString(region.EndKey),
			))
		}

		lastRegion = region
		lastEndKey = region.EndKey
	}
	if len(lastEndKey) != 0 {
		return errors.New(fmt.Sprintf("Last end key '%s' of region '%d' isn't empty",
			hex.EncodeToString(lastEndKey), lastRegion.Id,
		))
	}
	return nil
}

func (c *clusterInstance) startStores(address string) error {
	for _, store := range c.stores {
		if err := store.start(c.server, c, address); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterInstance) getStore(storeID uint64) (*metapb.Store, error) {
	c.RLock()
	defer c.RUnlock()
	if storeID == 0 {
		return nil, errors.New("invalid zero store id")
	}
	for _, store := range c.stores {
		if store.Id == storeID {
			return proto.Clone(&store.Store).(*metapb.Store), nil
		}
	}
	return nil, errors.Errorf("invalid store ID %d, not found", storeID)
}

func (c *clusterInstance) getStores() []*metapb.Store {
	c.RLock()
	defer c.RUnlock()
	stores := make([]*metapb.Store, len(c.stores))
	for idx, store := range c.stores {
		stores[idx] = proto.Clone(&store.Store).(*metapb.Store)
	}
	return stores
}

func (c *clusterInstance) initRegions() error {
	numStores := len(c.stores)
	for i, region := range c.regions {
		if err := region.init(c.server, c.stores[i%numStores]); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterInstance) start(address string) (err error) {
	defer func() {
		if err != nil {
			c.stop()
		}
	}()
	if err = c.member.start(address); err != nil {
		return err
	}
	if err = c.startStores(address); err != nil {
		return err
	}
	if err = c.initRegions(); err != nil {
		return err
	}

	if c.store, err = NewMVCCLevelDB(""); err != nil {
		return err
	}

	return
}

func (c *clusterInstance) stopStores() {
	for _, store := range c.stores {
		store.stop()
	}
}

func (c *clusterInstance) stop() {
	c.Lock()
	defer c.Unlock()
	c.stopStores()
	c.member.stop()
}

func (c *clusterInstance) validateRequest(header *pdpb.RequestHeader) error {
	if header.GetClusterId() != c.clusterID {
		return status.Errorf(codes.FailedPrecondition, "mismatch cluster id, need %d but got %d", c.clusterID, header.GetClusterId())
	}
	return nil
}

func (c *clusterInstance) getTS(count uint32) (physical, logical int64) {
	return c.server.getTS(count)
}

func (c *clusterInstance) header() *pdpb.ResponseHeader {
	return &pdpb.ResponseHeader{
		ClusterId: c.clusterID,
	}
}

func (c *clusterInstance) allocID() uint64 {
	return c.server.allocID()
}

func (c *clusterInstance) getRegionByKey(key []byte) (*metapb.Region, *metapb.Peer) {
	c.RLock()
	defer c.RUnlock()
	for _, r := range c.regions {
		if regionContains(r.StartKey, r.EndKey, key) {
			return proto.Clone(&r.Region).(*metapb.Region), proto.Clone(r.leaderPeer()).(*metapb.Peer)
		}
	}
	return nil, nil
}

func (c *clusterInstance) getRegionByID(regionID uint64) (*metapb.Region, *metapb.Peer) {
	c.RLock()
	defer c.RUnlock()
	for _, r := range c.regions {
		if r.GetId() == regionID {
			return proto.Clone(&r.Region).(*metapb.Region), proto.Clone(r.leaderPeer()).(*metapb.Peer)
		}
	}
	return nil, nil
}

func (c *clusterInstance) getConfig() *metapb.Cluster {
	c.RLock()
	defer c.RUnlock()
	return &metapb.Cluster{
		Id:           c.clusterID,
		MaxPeerCount: c.maxPeerCount,
	}
}

func (c *clusterInstance) setConfig(config *metapb.Cluster) {
	c.Lock()
	defer c.Unlock()
	c.maxPeerCount = config.MaxPeerCount
}

func (c *clusterInstance) getSafePoint() uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.safePoint
}

func (c *clusterInstance) updateSafePoint(safePoint uint64) {
	c.Lock()
	defer c.Unlock()
	if safePoint > c.safePoint {
		log.Infof("updated gc safe point to %d", safePoint)
		c.safePoint = safePoint
	} else if safePoint < c.safePoint {
		log.Warnf("trying to update gc safe point from %d to %d", c.safePoint, safePoint)
	}
	return
}
