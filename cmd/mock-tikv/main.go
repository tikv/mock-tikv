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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pingcap/log"
	"github.com/pkg/errors"
	"github.com/tikv/mock-tikv/server"
)

func main() {
	cfg := server.NewConfig()
	err := cfg.Parse(os.Args[1:])

	defer server.LogPanic()

	switch errors.Cause(err) {
	case nil:
	case flag.ErrHelp:
		os.Exit(0)
	default:
		log.S().Fatalf("parse cmd flags error: %s\n", fmt.Sprintf("%+v", err))
	}

	err = server.InitLogger(&cfg.Log)
	if err != nil {
		log.S().Fatalf("initialize logger error: %s\n", fmt.Sprintf("%+v", err))
	}

	_, err = server.CreateServer(cfg)
	if err != nil {
		log.S().Fatalf("create server failed: %v", fmt.Sprintf("%+v", err))
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	var sig os.Signal
	go func() {
		sig = <-sc
		cancel()
	}()

	<-ctx.Done()
	log.S().Infof("Got signal [%d] to exit.", sig)

	switch sig {
	case syscall.SIGTERM:
		os.Exit(0)
	default:
		os.Exit(1)
	}
}
