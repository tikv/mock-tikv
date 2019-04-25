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
	"context"
	"fmt"
	"io"
	"net"

	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memberInstance struct {
	pdpb.Member
	listener net.Listener
	cluster  *clusterInstance
	server   *grpc.Server
}

func newMemberInstance(idAlloc idAllocator, cluster *clusterInstance) *memberInstance {
	id := idAlloc.allocID()
	return &memberInstance{
		Member: pdpb.Member{
			MemberId:   id,
			Name:       fmt.Sprintf("mock_pd_%d", id),
			PeerUrls:   []string{},
			ClientUrls: []string{},
		},
		cluster: cluster,
	}
}

func (m *memberInstance) start(address string) error {
	listener, err := net.Listen("tcp", address+":0")
	if err != nil {
		return err
	}
	m.listener = listener
	m.ClientUrls = []string{"http://" + listener.Addr().String()}
	m.server = grpc.NewServer()
	pdpb.RegisterPDServer(m.server, m)
	go m.server.Serve(listener)
	return nil
}

func (m *memberInstance) stop() {
	m.server.GracefulStop()
}

func (m *memberInstance) GetMembers(ctx context.Context, request *pdpb.GetMembersRequest) (*pdpb.GetMembersResponse, error) {
	return &pdpb.GetMembersResponse{
		Header:     m.cluster.header(),
		Members:    []*pdpb.Member{&m.Member},
		Leader:     &m.Member,
		EtcdLeader: &m.Member,
	}, nil
}

func (m *memberInstance) Tso(stream pdpb.PD_TsoServer) error {
	for {
		request, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.WithStack(err)
		}
		if err = m.cluster.validateRequest(request.GetHeader()); err != nil {
			return err
		}
		count := request.GetCount()
		physical, logical := m.cluster.getTS(count)
		response := &pdpb.TsoResponse{
			Header: m.cluster.header(),
			Timestamp: &pdpb.Timestamp{
				Physical: physical,
				Logical:  logical,
			},
			Count: count,
		}
		if err := stream.Send(response); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (m *memberInstance) Bootstrap(ctx context.Context, request *pdpb.BootstrapRequest) (*pdpb.BootstrapResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}
	return &pdpb.BootstrapResponse{
		Header: m.cluster.header(),
	}, nil
}

func (m *memberInstance) IsBootstrapped(ctx context.Context, request *pdpb.IsBootstrappedRequest) (*pdpb.IsBootstrappedResponse, error) {
	return &pdpb.IsBootstrappedResponse{
		Header:       m.cluster.header(),
		Bootstrapped: true,
	}, nil
}

func (m *memberInstance) AllocID(ctx context.Context, request *pdpb.AllocIDRequest) (*pdpb.AllocIDResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	return &pdpb.AllocIDResponse{
		Header: m.cluster.header(),
		Id:     m.cluster.allocID(),
	}, nil
}

func (m *memberInstance) GetStore(ctx context.Context, request *pdpb.GetStoreRequest) (*pdpb.GetStoreResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	store, err := m.cluster.getStore(request.GetStoreId())
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &pdpb.GetStoreResponse{
		Header: m.cluster.header(),
		Store:  store,
	}, nil
}

func (m *memberInstance) PutStore(ctx context.Context, request *pdpb.PutStoreRequest) (*pdpb.PutStoreResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Put store is not implemented by mock pd server")
}

func (m *memberInstance) GetAllStores(ctx context.Context, request *pdpb.GetAllStoresRequest) (*pdpb.GetAllStoresResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	return &pdpb.GetAllStoresResponse{
		Header: m.cluster.header(),
		Stores: m.cluster.getStores(),
	}, nil
}

func (m *memberInstance) StoreHeartbeat(ctx context.Context, request *pdpb.StoreHeartbeatRequest) (*pdpb.StoreHeartbeatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Store heartbeat is not implemented by mock pd server")
}

func (m *memberInstance) RegionHeartbeat(stream pdpb.PD_RegionHeartbeatServer) error {
	return status.Errorf(codes.Unimplemented, "Put store is not implemented by mock pd server")
}

func (m *memberInstance) GetRegion(ctx context.Context, request *pdpb.GetRegionRequest) (*pdpb.GetRegionResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	region, leader := m.cluster.getRegionByKey(request.GetRegionKey())
	return &pdpb.GetRegionResponse{
		Header: m.cluster.header(),
		Region: region,
		Leader: leader,
	}, nil
}

func (m *memberInstance) GetPrevRegion(ctx context.Context, request *pdpb.GetRegionRequest) (*pdpb.GetRegionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetPrevRegion is not implemented by mock pd server")
}

func (m *memberInstance) GetRegionByID(ctx context.Context, request *pdpb.GetRegionByIDRequest) (*pdpb.GetRegionResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	region, leader := m.cluster.getRegionByID(request.GetRegionId())
	return &pdpb.GetRegionResponse{
		Header: m.cluster.header(),
		Region: region,
		Leader: leader,
	}, nil
}

func (m *memberInstance) AskSplit(ctx context.Context, request *pdpb.AskSplitRequest) (*pdpb.AskSplitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Ask split is not implemented by mock pd server")
}

func (m *memberInstance) ReportSplit(ctx context.Context, request *pdpb.ReportSplitRequest) (*pdpb.ReportSplitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Report split is not implemented by mock pd server")
}

func (m *memberInstance) AskBatchSplit(ctx context.Context, request *pdpb.AskBatchSplitRequest) (*pdpb.AskBatchSplitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Ask batch split is not implemented by mock pd server")
}

func (m *memberInstance) ReportBatchSplit(ctx context.Context, request *pdpb.ReportBatchSplitRequest) (*pdpb.ReportBatchSplitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Report batch split is not implemented by mock pd server")
}

func (m *memberInstance) GetClusterConfig(ctx context.Context, request *pdpb.GetClusterConfigRequest) (*pdpb.GetClusterConfigResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	return &pdpb.GetClusterConfigResponse{
		Header:  m.cluster.header(),
		Cluster: m.cluster.getConfig(),
	}, nil
}

func (m *memberInstance) PutClusterConfig(ctx context.Context, request *pdpb.PutClusterConfigRequest) (*pdpb.PutClusterConfigResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	m.cluster.setConfig(request.GetCluster())
	return &pdpb.PutClusterConfigResponse{
		Header: m.cluster.header(),
	}, nil
}

func (m *memberInstance) ScatterRegion(ctx context.Context, request *pdpb.ScatterRegionRequest) (*pdpb.ScatterRegionResponse, error) {
	return nil, nil
}

func (m *memberInstance) GetGCSafePoint(ctx context.Context, request *pdpb.GetGCSafePointRequest) (*pdpb.GetGCSafePointResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	return &pdpb.GetGCSafePointResponse{
		Header:    m.cluster.header(),
		SafePoint: m.cluster.getSafePoint(),
	}, nil
}

func (m *memberInstance) UpdateGCSafePoint(ctx context.Context, request *pdpb.UpdateGCSafePointRequest) (*pdpb.UpdateGCSafePointResponse, error) {
	if err := m.cluster.validateRequest(request.GetHeader()); err != nil {
		return nil, err
	}

	m.cluster.updateSafePoint(request.SafePoint)

	return &pdpb.UpdateGCSafePointResponse{
		Header:       m.cluster.header(),
		NewSafePoint: request.SafePoint,
	}, nil
}

func (m *memberInstance) SyncRegions(stream pdpb.PD_SyncRegionsServer) error {
	return status.Errorf(codes.Unimplemented, "SyncRegions is not implemented by mock pd server")
}
