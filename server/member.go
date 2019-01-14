package server

import (
	"fmt"
	"net"

	"github.com/pingcap/kvproto/pkg/pdpb"
)

type memberInstance struct {
	pdpb.Member
	listener net.Listener
}

func newMemberInstance(idAlloc idAllocator) *memberInstance {
	id := idAlloc.allocID()
	return &memberInstance{
		Member: pdpb.Member{
			MemberId:   id,
			Name:       fmt.Sprintf("mock_pd_%d", id),
			PeerUrls:   []string{},
			ClientUrls: []string{},
		},
	}
}

func (m *memberInstance) start(address string) error {
	listener, err := net.Listen("tcp", address+":0")
	if err != nil {
		return err
	}
	m.listener = listener
	m.ClientUrls = []string{listener.Addr().String()}
	return nil
}
