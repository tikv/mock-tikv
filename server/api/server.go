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
	"net/http"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

const apiPrefix = "/mock-tikv"

// Server defines public surface between internal implementation and api
type Server interface {
	GetCluster(id uint64) *Cluster
	GetClusters() []*Cluster
	DeleteCluster(id uint64)
	CreateCluster(cluster *Cluster) (*Cluster, error)
	GetClusterStore(clusterID, storeID uint64) (*Store, error)
	GetClusterStores(clusterID uint64) ([]*Store, error)
	GetStoreFailPoints(clusterID, storeID uint64) (map[string]interface{}, error)
	GetStoreFailPoint(clusterID, storeID uint64, failPoint string) (interface{}, error)
	UpdateStoreFailPoint(clusterID, storeID uint64, failPoint, value string) (interface{}, error)
	DeleteStoreFailPoint(clusterID, storeID uint64, failPoint string) (interface{}, error)
}

// NewHandler create s a HTTP handler for API
func NewHandler(server Server) http.Handler {
	engine := negroni.New()

	recovery := negroni.NewRecovery()
	engine.Use(recovery)

	router := mux.NewRouter()
	router.PathPrefix(apiPrefix).Handler(negroni.New(
		negroni.Wrap(createRouter(apiPrefix, server)),
	))

	engine.UseHandler(router)

	return engine
}
