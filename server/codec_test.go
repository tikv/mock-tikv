// Copyright 2015 PingCAP, Inc.
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
	"math"
	"testing"

	. "github.com/pingcap/check"
)

func TestT(t *testing.T) {
	CustomVerboseFlag = true
	TestingT(t)
}

var _ = Suite(&testCodecSuite{})

type testCodecSuite struct {
}

func (s *testCodecSuite) TestBytesCodec(c *C) {
	inputs := []struct {
		enc []byte
		dec []byte
	}{
		{[]byte{}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 247}},
		{[]byte{0}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 248}},
		{[]byte{1, 2, 3}, []byte{1, 2, 3, 0, 0, 0, 0, 0, 250}},
		{[]byte{1, 2, 3, 0}, []byte{1, 2, 3, 0, 0, 0, 0, 0, 251}},
		{[]byte{1, 2, 3, 4, 5, 6, 7}, []byte{1, 2, 3, 4, 5, 6, 7, 0, 254}},
		{[]byte{0, 0, 0, 0, 0, 0, 0, 0}, []byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 0, 0, 0, 0, 0, 0, 0, 0, 247}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, []byte{1, 2, 3, 4, 5, 6, 7, 8, 255, 0, 0, 0, 0, 0, 0, 0, 0, 247}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, []byte{1, 2, 3, 4, 5, 6, 7, 8, 255, 9, 0, 0, 0, 0, 0, 0, 0, 248}},
	}

	for _, input := range inputs {
		b := EncodeBytes(nil, input.enc)
		c.Assert(b, BytesEquals, input.dec)
		_, d, err := DecodeBytes(b, nil)
		c.Assert(err, IsNil)
		c.Assert(d, BytesEquals, input.enc)
	}

	// Test error decode.
	errInputs := [][]byte{
		{1, 2, 3, 4},
		{0, 0, 0, 0, 0, 0, 0, 247},
		{0, 0, 0, 0, 0, 0, 0, 0, 246},
		{0, 0, 0, 0, 0, 0, 0, 1, 247},
		{1, 2, 3, 4, 5, 6, 7, 8, 0},
		{1, 2, 3, 4, 5, 6, 7, 8, 255, 1},
		{1, 2, 3, 4, 5, 6, 7, 8, 255, 1, 2, 3, 4, 5, 6, 7, 8},
		{1, 2, 3, 4, 5, 6, 7, 8, 255, 1, 2, 3, 4, 5, 6, 7, 8, 255},
		{1, 2, 3, 4, 5, 6, 7, 8, 255, 1, 2, 3, 4, 5, 6, 7, 8, 0},
	}

	for _, input := range errInputs {
		_, _, err := DecodeBytes(input, nil)
		c.Assert(err, NotNil)
	}
}

func (s *testCodecSuite) TestNumberCodec(c *C) {
	tblUint64 := []uint64{
		0,
		math.MaxUint8,
		math.MaxUint16,
		math.MaxUint32,
		math.MaxUint64,
		1<<24 - 1,
		1<<48 - 1,
		1<<56 - 1,
		1,
		math.MaxInt16,
		math.MaxInt8,
		math.MaxInt32,
		math.MaxInt64,
	}

	for _, t := range tblUint64 {
		b := EncodeUintDesc(nil, t)
		_, v, err := DecodeUintDesc(b)
		c.Assert(err, IsNil)
		c.Assert(v, Equals, t)
	}
}

func (s *testCodecSuite) TestNumberOrder(c *C) {
	tblUint64 := []struct {
		Arg1 uint64
		Arg2 uint64
		Ret  int
	}{
		{0, 0, 0},
		{1, 0, 1},
		{0, 1, -1},
		{math.MaxInt8, math.MaxInt16, -1},
		{math.MaxUint32, math.MaxInt32, 1},
		{math.MaxUint8, math.MaxInt8, 1},
		{math.MaxUint16, math.MaxInt32, -1},
		{math.MaxUint64, math.MaxInt64, 1},
		{math.MaxInt64, math.MaxUint32, 1},
		{math.MaxUint64, 0, 1},
		{0, math.MaxUint64, -1},
	}

	for _, t := range tblUint64 {
		b1 := EncodeUintDesc(nil, t.Arg1)
		b2 := EncodeUintDesc(nil, t.Arg2)

		ret := bytes.Compare(b1, b2)
		c.Assert(ret, Equals, -t.Ret)
	}
}
