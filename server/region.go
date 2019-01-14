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
		&metapb.Peer{
			Id:      idAlloc.allocID(),
			StoreId: store.Id,
		},
	}
	return nil
}
