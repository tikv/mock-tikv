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
	"github.com/gorilla/mux"
	"github.com/unrolled/render"
)

func createRouter(prefix string, server Server) *mux.Router {
	rd := render.New(render.Options{
		IndentJSON: true,
	})

	router := mux.NewRouter().PathPrefix(prefix).Subrouter()

	clusterHandler := newClusterHandler(server, rd)
	router.HandleFunc("/api/v1/clusters", clusterHandler.List).Methods("GET")
	router.HandleFunc("/api/v1/clusters", clusterHandler.Post).Methods("POST")
	router.HandleFunc("/api/v1/clusters/{cluster_id}", clusterHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/clusters/{cluster_id}", clusterHandler.Delete).Methods("DELETE")

	storeHandler := newStoreHandler(server, rd)
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores", storeHandler.List).Methods("GET")
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores/{store_id}", storeHandler.Get).Methods("GET")
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores/{store_id}/failpoints", storeHandler.ListFailPoints).Methods("GET")
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores/{store_id}/failpoints/{failpoint}", storeHandler.UpdateFailPoint).Methods("POST", "PUT")
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores/{store_id}/failpoints/{failpoint}", storeHandler.GetFailPoint).Methods("GET")
	router.HandleFunc("/api/v1/clusters/{cluster_id}/stores/{store_id}/failpoints/{failpoint}", storeHandler.DeleteFailPoint).Methods("DELETE")
	return router
}
