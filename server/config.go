package server

import (
	"flag"

	"github.com/pingcap/pd/pkg/logutil"
	"github.com/pingcap/pd/pkg/metricutil"
)

const (
	defaultClientEndpoint = "127.0.0.1:2378"
)

// Config is the mock tikv server configuration
type Config struct {
	*flag.FlagSet `json:"-"`

	ClientEndpoint string `toml:"client-endpoint" json:"client-endpoint"`

	// Log related config.
	Log logutil.LogConfig `toml:"log" json:"log"`

	// Metric related config.
	Metric metricutil.MetricConfig `toml:"metric" json:"metric"`

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
