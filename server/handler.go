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
