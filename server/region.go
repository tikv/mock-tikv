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
)

type regionInstance struct {
	metapb.Region
}

func newRegionInstance(id uint64, startKey, endKey []byte) *regionInstance {
	return &regionInstance{
		Region: metapb.Region{
			Id:       id,
			StartKey: startKey,
			EndKey:   endKey,
		},
	}
}

func (r *regionInstance) start(idAlloc idAllocator, store *storeInstance) error {
	r.Id = idAlloc.allocID()
	r.RegionEpoch = &metapb.RegionEpoch{
		ConfVer: idAlloc.allocID(),
		Version: idAlloc.allocID(),
	}
	r.Peers = []*metapb.Peer{
		{
			Id:      idAlloc.allocID(),
			StoreId: store.Id,
		},
	}
	return nil
}
