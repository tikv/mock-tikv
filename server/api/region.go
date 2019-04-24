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

import (
	"encoding/hex"
	"encoding/json"
)

// RegionEpoch records epoch data of a region
type RegionEpoch struct {
	ConfVer uint64 `json:"conf_ver"`
	Version uint64 `json:"version"`
}

// Region records region setup for api usage.
type Region struct {
	ID          uint64       `json:"id"`
	StartKey    []byte       `json:"-"`
	EndKey      []byte       `json:"-"`
	RegionEpoch *RegionEpoch `json:"region_epoch"`
	Peers       []*Peer      `json:"peers"`
}

// MarshalJSON marshal Region to json using unified hex key format
func (r *Region) MarshalJSON() ([]byte, error) {
	type Alias Region
	return json.Marshal(&struct {
		StartKey string `json:"start_key"`
		EndKey   string `json:"end_key"`
		*Alias
	}{
		StartKey: hex.EncodeToString(r.StartKey),
		EndKey:   hex.EncodeToString(r.EndKey),
		Alias:    (*Alias)(r),
	})
}

// UnmarshalJSON unmarshal Region from json with unified hex key format
func (r *Region) UnmarshalJSON(data []byte) error {
	type Alias Region
	aux := &struct {
		StartKey string `json:"start_key"`
		EndKey   string `json:"end_key"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	err := json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}
	if r.StartKey, err = hex.DecodeString(aux.StartKey); err != nil {
		return err
	}
	if r.EndKey, err = hex.DecodeString(aux.EndKey); err != nil {
		return err
	}
	return nil
}
