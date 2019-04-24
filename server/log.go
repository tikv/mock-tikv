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
	"os"
	"runtime/debug"
	"sync"

	"github.com/pingcap/log"
	"github.com/pkg/errors"
)

const (
	defaultLogMaxSize = 300 // MB
)

// InitFileLog initializes file based logging options.
func InitFileLog(cfg *log.FileLogConfig) error {
	if st, err := os.Stat(cfg.Filename); err == nil {
		if st.IsDir() {
			return errors.New("can't use directory as log file name")
		}
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = defaultLogMaxSize
	}

	return nil
}

var once sync.Once

// InitLogger initializes PD's logger.
func InitLogger(cfg *log.Config) error {
	var err error

	once.Do(func() {
		if len(cfg.File.Filename) == 0 {
			return
		}
		err = InitFileLog(&cfg.File)
	})
	return err
}

// LogPanic logs the panic reason and stack, then exit the process.
// Commonly used with a `defer`.
func LogPanic() {
	if e := recover(); e != nil {
		log.S().Fatalf("panic: %v, stack: %s", e, string(debug.Stack()))
	}
}
