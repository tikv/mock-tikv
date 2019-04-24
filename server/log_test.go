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
	"testing"

	. "github.com/pingcap/check"
	zaplog "github.com/pingcap/log"
)

const (
	logPattern = `\d\d\d\d/\d\d/\d\d \d\d:\d\d:\d\d\.\d\d\d ([\w_%!$@.,+~-]+|\\.)+:\d+: \[(fatal|error|warning|info|debug)\] .*?\n`
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testLogSuite{})

type testLogSuite struct{}

func (s *testLogSuite) SetUpSuite(c *C) {}

// TestLogging assure log format and log redirection works.
func (s *testLogSuite) TestLogging(c *C) {
	conf := &zaplog.Config{Level: "warn", File: zaplog.FileLogConfig{}}
	c.Assert(InitLogger(conf), IsNil)
}

func (s *testLogSuite) TestFileLog(c *C) {
	c.Assert(InitFileLog(&zaplog.FileLogConfig{Filename: "/tmp"}), NotNil)
	c.Assert(InitFileLog(&zaplog.FileLogConfig{Filename: "/tmp/test_file_log", MaxSize: 0}), IsNil)
}
