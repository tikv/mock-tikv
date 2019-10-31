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
	"github.com/tikv/mock-tikv/server/api"
)

// GetCluster returns current state of tikv cluster with given id
func (s *Server) GetCluster(id uint64) *api.Cluster {
	return convertClusterToAPI(s.doGetCluster(id))
}

// GetClusters returns all currently available tikv clusters
func (s *Server) GetClusters() []*api.Cluster {
	instances := s.doGetClusters()
	clusters := make([]*api.Cluster, len(instances))
	for i, instance := range instances {
		clusters[i] = convertClusterToAPI(instance)
	}
	return clusters
}

// DeleteCluster stops and deletes a cluster by id
func (s *Server) DeleteCluster(id uint64) {
	s.doDeleteCluster(id)
}

// CreateCluster adds and starts a new tikv cluster with given spec.
func (s *Server) CreateCluster(cluster *api.Cluster) (*api.Cluster, error) {
	instance, err := s.doCreateCluster(convertAPIToRegions(cluster.Regions), convertAPIToStores(cluster.Stores))
	if err != nil {
		return nil, err
	}
	return convertClusterToAPI(instance), nil
}

// GetClusterStore returns current state of tikv store of a cluster with given id
func (s *Server) GetClusterStore(clusterID, storeID uint64) (*api.Store, error) {
	cluster := s.doGetCluster(clusterID)
	if cluster == nil {
		return nil, errClusterNotFound
	}
	store, err := cluster.getStore(storeID)
	if err != nil {
		return nil, err
	}
	return convertStoreToAPI(store), nil
}

// GetClusterStores returns current state of tikv stores of a cluster
func (s *Server) GetClusterStores(clusterID uint64) ([]*api.Store, error) {
	cluster := s.doGetCluster(clusterID)
	if cluster == nil {
		return nil, errClusterNotFound
	}
	return convertStoresToAPI(cluster.getStores()), nil
}

// GetStoreFailPoints returns all fail points of a store
func (s *Server) GetStoreFailPoints(clusterID, storeID uint64) (map[string]interface{}, error) {
	cluster := s.doGetCluster(clusterID)
	if cluster == nil {
		return nil, errClusterNotFound
	}
	return cluster.getStoreFailPoints(storeID)
}

// GetStoreFailPoint returns given fail point of a store
func (s *Server) GetStoreFailPoint(clusterID, storeID uint64, failPoint string) (interface{}, error) {
	if failPoints, err := s.GetStoreFailPoints(clusterID, storeID); err != nil {
		return nil, err
	} else if fp, ok := failPoints[failPoint]; !ok {
		return nil, errFailPointNotFound
	} else {
		return fp, nil
	}
}

// UpdateStoreFailPoint Update value of a particular fail point
func (s *Server) UpdateStoreFailPoint(clusterID, storeID uint64, failPoint, value string) (interface{}, error) {
	cluster := s.doGetCluster(clusterID)
	if cluster == nil {
		return nil, errClusterNotFound
	}
	return cluster.updateStoreFailPoint(storeID, failPoint, value)
}

// DeleteStoreFailPoint Disable a particular fail point
func (s *Server) DeleteStoreFailPoint(clusterID, storeID uint64, failPoint string) (interface{}, error) {
	cluster := s.doGetCluster(clusterID)
	if cluster == nil {
		return nil, errClusterNotFound
	}
	return cluster.deleteStoreFailPoint(storeID, failPoint)
}
