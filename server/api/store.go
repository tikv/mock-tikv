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

// StoreState contains state about a store.
type StoreState string

const (
	// StoreStateUp store is up and running
	StoreStateUp StoreState = "Up"
	// StoreStateOffline store is offline
	StoreStateOffline StoreState = "Offline"
	// StoreStateTombstone store is set to tombstone state
	StoreStateTombstone StoreState = "Tombstone"
)

// Store contains information about a store
type Store struct {
	ID      uint64     `json:"id"`
	Address string     `json:"address"`
	State   StoreState `json:"state"`
	Version string     `json:"version"`
}
