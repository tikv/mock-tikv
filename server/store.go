package server

import (
	"net"

	"github.com/pingcap/kvproto/pkg/metapb"
)

type storeInstance struct {
	metapb.Store
	listener net.Listener
}

func newStoreInstance(version string) *storeInstance {
	return &storeInstance{
		Store: metapb.Store{
			Version: version,
		},
	}
}

func (s *storeInstance) start(idAlloc idAllocator, address string) error {
	listener, err := net.Listen("tcp", address+":0")
	if err != nil {
		return err
	}
	s.listener = listener
	s.Id = idAlloc.allocID()
	s.Address = listener.Addr().String()
	s.State = metapb.StoreState_Up
	return nil
}

func (s *storeInstance) stop() {
	s.listener.Close()
}
