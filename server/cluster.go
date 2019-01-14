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
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
)

const defaultTiKVVersion = "3.0.0"

var emptyKey = []byte{}

type clusterInstance struct {
	server    *Server
	clusterID uint64

	member  *memberInstance // we only support one leader pd at this time
	regions []*regionInstance
	stores  []*storeInstance

	idAlloc uint64
}

func (s *Server) doGetCluster(id uint64) *clusterInstance {
	if cluster, ok := s.clusters[id]; ok {
		return cluster
	}
	return nil
}

func (s *Server) doGetClusters() []*clusterInstance {
	result := make([]*clusterInstance, 0, len(s.clusters))
	for _, instance := range s.clusters {
		result = append(result, instance)
	}
	return result
}

func (s *Server) doDeleteCluster(id uint64) {
}

func validateRegionAndStore(regions []*regionInstance, stores []*storeInstance) error {
	var lastEndKey []byte = nil
	var lastRegion *regionInstance = nil
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
					base64.StdEncoding.EncodeToString(region.StartKey), region.Id,
					base64.StdEncoding.EncodeToString(lastEndKey), lastRegion.Id,
				))
			}
		}
		if len(region.EndKey) > 0 && len(region.StartKey) > 0 && bytes.Compare(region.StartKey, region.EndKey) >= 0 {
			return errors.New(fmt.Sprintf("Start key '%s' of region '%d' isn't the less than end key '%s'",
				base64.StdEncoding.EncodeToString(region.StartKey), region.Id,
				base64.StdEncoding.EncodeToString(region.EndKey),
			))
		}

		lastEndKey = region.EndKey
	}
	if len(lastEndKey) != 0 {
		return errors.New(fmt.Sprintf("Last end key '%s' of region '%d' isn't empty",
			base64.StdEncoding.EncodeToString(lastEndKey), lastRegion.Id,
		))
	}
	return nil
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
	if err := validateRegionAndStore(regions, stores); err != nil {
		return nil, err
	}
	instance := &clusterInstance{
		server:    s,
		clusterID: s.allocID(),
		member:    newMemberInstance(s),
		regions:   regions,
		stores:    stores,
	}
	if err := instance.start(s.address); err != nil {
		return nil, err
	}
	return instance, nil
}

func (c *clusterInstance) startStores(address string) error {
	for _, store := range c.stores {
		if err := store.start(c.server, address); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterInstance) startRegions() error {
	numStores := len(c.stores)
	for i, region := range c.regions {
		if err := region.start(c.server, c.stores[i%numStores]); err != nil {
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
	return c.startRegions()
}

func (c *clusterInstance) stopStores() {
}

func (c *clusterInstance) stop() {
	c.stopStores()
}
