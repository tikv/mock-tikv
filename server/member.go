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
