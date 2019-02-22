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
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

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

type storeHandler struct {
	Server
	r *render.Render
}

func newStoreHandler(server Server, r *render.Render) *storeHandler {
	return &storeHandler{
		Server: server,
		r:      r,
	}
}

func parseClusterAndStoreID(r *http.Request) (clusterID, storeID uint64, err error) {
	clusterID, err = parseClusterID(r)
	if err != nil {
		return 0, 0, err
	}

	storeID, err = parseID(r, "store_id")
	return clusterID, storeID, err
}

func (h *storeHandler) List(w http.ResponseWriter, r *http.Request) {
	clusterID, err := parseClusterID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if stores, err := h.GetClusterStores(clusterID); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, stores)
	}
}

func (h *storeHandler) Get(w http.ResponseWriter, r *http.Request) {
	clusterID, storeID, err := parseClusterAndStoreID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if store, err := h.GetClusterStore(clusterID, storeID); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, store)
	}
}

func (h *storeHandler) ListFailPoints(w http.ResponseWriter, r *http.Request) {
	clusterID, storeID, err := parseClusterAndStoreID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if failPoints, err := h.GetStoreFailPoints(clusterID, storeID); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, failPoints)
	}
}

func (h *storeHandler) GetFailPoint(w http.ResponseWriter, r *http.Request) {
	clusterID, storeID, err := parseClusterAndStoreID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if failPoints, err := h.GetStoreFailPoint(clusterID, storeID, mux.Vars(r)["failpoint"]); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, failPoints)
	}
}

func (h *storeHandler) DeleteFailPoint(w http.ResponseWriter, r *http.Request) {
	clusterID, storeID, err := parseClusterAndStoreID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if failPoint, err := h.DeleteStoreFailPoint(clusterID, storeID, mux.Vars(r)["failpoint"]); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, failPoint)
	}
}

func (h *storeHandler) UpdateFailPoint(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		h.r.JSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	clusterID, storeID, err := parseClusterAndStoreID(r)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if failPoint, err := h.UpdateStoreFailPoint(clusterID, storeID, mux.Vars(r)["failpoint"], string(data)); err != nil {
		h.r.JSON(w, http.StatusNotFound, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, failPoint)
	}
}
