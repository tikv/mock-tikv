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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

// Cluster contains all information about a mock tikv cluster
type Cluster struct {
	ID      uint64    `json:"id"`
	Regions []*Region `json:"regions"`
	Stores  []*Store  `json:"stores"`
	Members []*Member `json:"members"`
}

type clusterHandler struct {
	Server
	r *render.Render
}

func newClusterHandler(server Server, r *render.Render) *clusterHandler {
	return &clusterHandler{
		Server: server,
		r:      r,
	}
}

func (h *clusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["cluster_id"]
	clusterID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if cluster := h.GetCluster(clusterID); cluster == nil {
		h.r.JSON(w, http.StatusNotFound, "Could not find cluster")
	} else {
		h.r.JSON(w, http.StatusOK, cluster)
	}
}

func (h *clusterHandler) List(w http.ResponseWriter, r *http.Request) {
	clusters := h.GetClusters()
	if clusters == nil {
		clusters = make([]*Cluster, 0)
	}
	h.r.JSON(w, http.StatusOK, clusters)
}

func (h *clusterHandler) Post(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		h.r.JSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cluster Cluster
	if err := json.Unmarshal(data, &cluster); err != nil {
		h.r.JSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if resp, err := h.CreateCluster(&cluster); err != nil {
		h.r.JSON(w, http.StatusInternalServerError, err.Error())
	} else {
		h.r.JSON(w, http.StatusOK, resp)
	}
}

func (h *clusterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["cluster_id"]
	clusterID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	h.DeleteCluster(clusterID)
	h.r.JSON(w, http.StatusOK, nil)
}

func parseID(r *http.Request, name string) (uint64, error) {
	return strconv.ParseUint(mux.Vars(r)[name], 10, 64)
}

func parseClusterID(r *http.Request) (uint64, error) {
	return parseID(r, "cluster_id")
}
