// Copyright 2017 PingCAP, Inc.
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
	"bytes"
	"testing"

	. "github.com/pingcap/check"
	log "github.com/sirupsen/logrus"
)

const (
	logPattern = `\d\d\d\d/\d\d/\d\d \d\d:\d\d:\d\d\.\d\d\d  \[(fatal|error|warning|info|debug)\] .*?\n`
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testLogSuite{})

type testLogSuite struct {
	buf *bytes.Buffer
}

func (s *testLogSuite) SetUpSuite(c *C) {
	s.buf = &bytes.Buffer{}
}

func (s *testLogSuite) TestStringToLogLevel(c *C) {
	c.Assert(StringToLogLevel("fatal"), Equals, log.FatalLevel)
	c.Assert(StringToLogLevel("ERROR"), Equals, log.ErrorLevel)
	c.Assert(StringToLogLevel("warn"), Equals, log.WarnLevel)
	c.Assert(StringToLogLevel("warning"), Equals, log.WarnLevel)
	c.Assert(StringToLogLevel("debug"), Equals, log.DebugLevel)
	c.Assert(StringToLogLevel("info"), Equals, log.InfoLevel)
	c.Assert(StringToLogLevel("whatever"), Equals, log.InfoLevel)
}

func (s *testLogSuite) TestStringToLogFormatter(c *C) {
	c.Assert(StringToLogFormatter("text", true), DeepEquals, &textFormatter{
		DisableTimestamp: true,
	})
	c.Assert(StringToLogFormatter("json", true), DeepEquals, &log.JSONFormatter{
		DisableTimestamp: true,
		TimestampFormat:  defaultLogTimeFormat,
	})
	c.Assert(StringToLogFormatter("console", true), DeepEquals, &log.TextFormatter{
		DisableTimestamp: true,
		FullTimestamp:    true,
		TimestampFormat:  defaultLogTimeFormat,
	})
	c.Assert(StringToLogFormatter("", true), DeepEquals, &textFormatter{})
}

// TestLogging assure log format and log redirection works.
func (s *testLogSuite) TestLogging(c *C) {
	conf := &LogConfig{Level: "warn", File: FileLogConfig{}}
	c.Assert(InitLogger(conf), IsNil)

	log.SetOutput(s.buf)
	log.Warnf("this message comes from logrus")
	entry, err := s.buf.ReadString('\n')
	c.Assert(err, IsNil)
	c.Assert(entry, Matches, logPattern)
}

func (s *testLogSuite) TestFileLog(c *C) {
	c.Assert(InitFileLog(&FileLogConfig{Filename: "/tmp"}), NotNil)
	c.Assert(InitFileLog(&FileLogConfig{Filename: "/tmp/test_file_log", MaxSize: 0}), IsNil)
}
