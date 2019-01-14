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
