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

package api

// RegionEpoch records epoch data of a region
type RegionEpoch struct {
	ConfVer uint64 `json:"conf_ver"`
	Version uint64 `json:"version"`
}

// Region records region setup for api usage.
type Region struct {
	ID          uint64       `json:"id"`
	StartKey    []byte       `json:"start_key"`
	EndKey      []byte       `json:"end_key"`
	RegionEpoch *RegionEpoch `json:"region_epoch"`
	Peers       []*Peer      `json:"peers"`
}
