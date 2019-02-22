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
	"flag"

	"github.com/pingcap/log"
)

const (
	defaultClientEndpoint = "127.0.0.1:2378"
)

// Config is the mock tikv server configuration
type Config struct {
	*flag.FlagSet `json:"-"`

	ClientEndpoint string `toml:"client-endpoint" json:"client-endpoint"`

	// Log related config.
	Log log.Config `toml:"log" json:"log"`

	configFile string
}

// NewConfig creates a new config.
func NewConfig() *Config {
	cfg := &Config{}
	cfg.FlagSet = flag.NewFlagSet("mock-tikv", flag.ContinueOnError)
	fs := cfg.FlagSet

	fs.StringVar(&cfg.configFile, "config", "", "Config file")

	fs.StringVar(&cfg.ClientEndpoint, "client-endpoint", defaultClientEndpoint, "endpoint for client traffic")

	fs.StringVar(&cfg.Log.Level, "L", "", "log level: debug, info, warn, error, fatal (default 'info')")
	fs.StringVar(&cfg.Log.File.Filename, "log-file", "", "log file path")
	fs.BoolVar(&cfg.Log.File.LogRotate, "log-rotate", true, "rotate log")

	return cfg
}
