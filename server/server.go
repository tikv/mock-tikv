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
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tikv/mock-tikv/server/api"
)

// Server is the mock tikv server.
type Server struct {
	// Configs and initial fields.
	cfg *Config

	address string
	ctx     context.Context
	cancel  context.CancelFunc

	idAlloc  uint64
	clusters map[uint64]*clusterInstance
}

// CreateServer creates the mock tikv server with given configuration.
func CreateServer(cfg *Config) (s *Server, err error) {
	log.Infof("Mock TiKV config - %v", cfg)
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

type idAllocator interface {
	allocID() uint64
}
