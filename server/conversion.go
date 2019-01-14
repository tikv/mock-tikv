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
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/tikv/mock-tikv/server/api"
)

func convertMemberToAPI(member *memberInstance) []*api.Member {
	return []*api.Member{
		&api.Member{
			Name:       member.Name,
			MemberID:   member.MemberId,
			ClientUrls: member.ClientUrls,
		},
	}
}

func convertClusterToAPI(instance *clusterInstance) *api.Cluster {
	if instance == nil {
		return nil
	}
	return &api.Cluster{
		ID:      instance.clusterID,
		Members: convertMemberToAPI(instance.member),
		Regions: convertRegionsToAPI(instance.regions),
		Stores:  convertStoresToAPI(instance.stores),
	}
}

func convertPeerToAPI(peer *metapb.Peer) *api.Peer {
	return &api.Peer{
		ID:      peer.Id,
		StoreID: peer.StoreId,
	}
}

func convertPeersToAPI(peers []*metapb.Peer) []*api.Peer {
	result := make([]*api.Peer, len(peers))
	for i, peer := range peers {
		result[i] = convertPeerToAPI(peer)
	}
	return result
}

func convertRegionEpochToAPI(epoch *metapb.RegionEpoch) *api.RegionEpoch {
	return &api.RegionEpoch{
		ConfVer: epoch.ConfVer,
		Version: epoch.Version,
	}
}

func convertAPIToRegion(region *api.Region) *regionInstance {
	return newRegionInstance(region.ID, region.StartKey, region.EndKey)
}

func convertAPIToRegions(regions []*api.Region) []*regionInstance {
	result := make([]*regionInstance, len(regions))
	for i, region := range regions {
		result[i] = convertAPIToRegion(region)
	}
	return result
}

func convertRegionToAPI(region *regionInstance) *api.Region {
	return &api.Region{
		ID:          region.Id,
		StartKey:    region.StartKey,
		EndKey:      region.EndKey,
		RegionEpoch: convertRegionEpochToAPI(region.RegionEpoch),
		Peers:       convertPeersToAPI(region.Peers),
	}
}

func convertRegionsToAPI(regions []*regionInstance) []*api.Region {
	result := make([]*api.Region, len(regions))
	for i, region := range regions {
		result[i] = convertRegionToAPI(region)
	}
	return result
}

func convertAPIToStore(store *api.Store) *storeInstance {
	return &storeInstance{}
}

func convertAPIToStores(stores []*api.Store) []*storeInstance {
	result := make([]*storeInstance, len(stores))
	for i, store := range stores {
		result[i] = convertAPIToStore(store)
	}
	return result
}

func convertStoreStateToAPI(state metapb.StoreState) api.StoreState {
	switch state {
	case metapb.StoreState_Up:
		return api.StoreStateUp
	case metapb.StoreState_Tombstone:
		return api.StoreStateTombstone
	case metapb.StoreState_Offline:
		fallthrough
	default:
		return api.StoreStateOffline
	}
}

func convertStoreToAPI(store *storeInstance) *api.Store {
	return &api.Store{
		ID:      store.Id,
		Address: store.Address,
		State:   convertStoreStateToAPI(store.State),
		Version: store.Version,
	}
}

func convertStoresToAPI(stores []*storeInstance) []*api.Store {
	result := make([]*api.Store, len(stores))
	for i, store := range stores {
		result[i] = convertStoreToAPI(store)
	}
	return result
}
