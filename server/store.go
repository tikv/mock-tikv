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
	"bytes"
	"context"
	"github.com/pingcap/failpoint"
	"io"
	"net"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/tikvpb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const requestMaxSize = 8 * 1024 * 1024

const (
	serverBusyFailPoint = "server-busy"
)

type storeRequestContext struct {
	store *MVCCLevelDB
	// store id for current request
	storeID uint64
	// Used for handling normal request.
	startKey []byte
	endKey   []byte
	// Used for handling coprocessor request.
	rawStartKey []byte
	rawEndKey   []byte
	// Used for current request.
	isolationLevel kvrpcpb.IsolationLevel
}

func (c *storeRequestContext) checkKeyInRegion(key []byte) bool {
	return regionContains(c.startKey, c.endKey, []byte(NewMvccKey(key)))
}

type storeInstance struct {
	metapb.Store
	listener net.Listener

	cluster *clusterInstance
	server  *grpc.Server

	ctx context.Context

	rpcCommitResult  string
	rpcCommitTimeout bool
}

func newStoreInstance(version string) *storeInstance {
	return &storeInstance{
		Store: metapb.Store{
			Version: version,
		},
	}
}

func (s *storeInstance) start(cluster *clusterInstance, address string) error {
	listener, err := net.Listen("tcp", address+":0")
	if err != nil {
		return err
	}
	s.listener = listener
	s.cluster = cluster
	s.Address = listener.Addr().String()
	s.State = metapb.StoreState_Up
	s.server = grpc.NewServer()
	tikvpb.RegisterTikvServer(s.server, s)
	go s.server.Serve(s.listener)
	return nil
}

func (s *storeInstance) stop() {
	s.server.GracefulStop()
}

func convertToKeyError(err error) *kvrpcpb.KeyError {
	if locked, ok := errors.Cause(err).(*ErrLocked); ok {
		return &kvrpcpb.KeyError{
			Locked: &kvrpcpb.LockInfo{
				Key:         locked.Key.Raw(),
				PrimaryLock: locked.Primary,
				LockVersion: locked.StartTS,
				LockTtl:     locked.TTL,
			},
		}
	}
	if retryable, ok := errors.Cause(err).(ErrRetryable); ok {
		return &kvrpcpb.KeyError{
			Retryable: retryable.Error(),
		}
	}
	return &kvrpcpb.KeyError{
		Abort: err.Error(),
	}
}

func convertToKeyErrors(errs []error) []*kvrpcpb.KeyError {
	var keyErrors = make([]*kvrpcpb.KeyError, 0)
	for _, err := range errs {
		if err != nil {
			keyErrors = append(keyErrors, convertToKeyError(err))
		}
	}
	return keyErrors
}

func convertToPbPairsKeyOnly(pairs []Pair) []*kvrpcpb.KvPair {
	kvPairs := convertToPbPairs(pairs)
	for _, pair := range kvPairs {
		pair.Value = nil
	}
	return kvPairs
}

func convertToPbPairs(pairs []Pair) []*kvrpcpb.KvPair {
	kvPairs := make([]*kvrpcpb.KvPair, 0, len(pairs))
	for _, p := range pairs {
		var kvPair *kvrpcpb.KvPair
		if p.Err == nil {
			kvPair = &kvrpcpb.KvPair{
				Key:   p.Key,
				Value: p.Value,
			}
		} else {
			kvPair = &kvrpcpb.KvPair{
				Error: convertToKeyError(p.Err),
			}
		}
		kvPairs = append(kvPairs, kvPair)
	}
	return kvPairs
}

func (s *storeInstance) checkRequestSize(size int) *errorpb.Error {
	// TiKV has a limitation on raft log size.
	// mocktikv has no raft inside, so we check the request's size instead.
	if size >= requestMaxSize {
		return &errorpb.Error{
			RaftEntryTooLarge: &errorpb.RaftEntryTooLarge{},
		}
	}
	return nil
}

func (s *storeInstance) checkRequestContext(ctx *kvrpcpb.Context) (*storeRequestContext, *errorpb.Error) {
	ctxPeer := ctx.GetPeer()
	if ctxPeer != nil && ctxPeer.GetStoreId() != s.Id {
		return nil, &errorpb.Error{
			Message:       *proto.String("store not match"),
			StoreNotMatch: &errorpb.StoreNotMatch{},
		}
	}
	region, leader := s.cluster.getRegionByID(ctx.GetRegionId())
	// No region found.
	if region == nil {
		return nil, &errorpb.Error{
			Message: *proto.String("region not found"),
			RegionNotFound: &errorpb.RegionNotFound{
				RegionId: *proto.Uint64(ctx.GetRegionId()),
			},
		}
	}
	leaderID := leader.Id
	var storePeer, leaderPeer *metapb.Peer
	for _, p := range region.Peers {
		if p.GetStoreId() == s.Id {
			storePeer = p
		}
		if p.GetId() == leaderID {
			leaderPeer = p
		}
	}
	// The store does not contain a Peer of the Region.
	if storePeer == nil {
		return nil, &errorpb.Error{
			Message: *proto.String("region not found"),
			RegionNotFound: &errorpb.RegionNotFound{
				RegionId: *proto.Uint64(ctx.GetRegionId()),
			},
		}
	}
	// No leader.
	if leaderPeer == nil {
		return nil, &errorpb.Error{
			Message: *proto.String("no leader"),
			NotLeader: &errorpb.NotLeader{
				RegionId: *proto.Uint64(ctx.GetRegionId()),
			},
		}
	}
	// The Peer on the Store is not leader.
	if storePeer.GetId() != leaderPeer.GetId() {
		return nil, &errorpb.Error{
			Message: *proto.String("not leader"),
			NotLeader: &errorpb.NotLeader{
				RegionId: *proto.Uint64(ctx.GetRegionId()),
				Leader:   leaderPeer,
			},
		}
	}
	// Region epoch does not match.
	if !proto.Equal(region.GetRegionEpoch(), ctx.GetRegionEpoch()) {
		nextRegion, _ := s.cluster.getRegionByKey(region.GetEndKey())
		currentRegions := []*metapb.Region{region}
		if nextRegion != nil {
			currentRegions = append(currentRegions, nextRegion)
		}
		return nil, &errorpb.Error{
			Message: *proto.String("epoch not match"),
			EpochNotMatch: &errorpb.EpochNotMatch{
				CurrentRegions: currentRegions,
			},
		}
	}
	return &storeRequestContext{
		store:          s.cluster.store,
		storeID:        s.Id,
		startKey:       region.StartKey,
		endKey:         region.EndKey,
		isolationLevel: ctx.GetIsolationLevel(),
	}, nil
}

func (s *storeInstance) checkRequest(ctx *kvrpcpb.Context, size int) (*storeRequestContext, *errorpb.Error) {
	failpoint.InjectContext(s.ctx, serverBusyFailPoint, func() {
		failpoint.Return(nil, &errorpb.Error{ServerIsBusy: &errorpb.ServerIsBusy{}})
	})
	reqCtx, regionErr := s.checkRequestContext(ctx)
	if regionErr != nil {
		return nil, regionErr
	}
	return reqCtx, s.checkRequestSize(size)
}

// KV commands with mvcc/txn supported.
func (s *storeInstance) KvGet(ctx context.Context, req *kvrpcpb.GetRequest) (*kvrpcpb.GetResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.GetResponse{RegionError: regionErr}, nil
	}
	val, err := reqCtx.store.Get(req.GetKey(), req.GetVersion(), reqCtx.isolationLevel)
	if err != nil {
		return &kvrpcpb.GetResponse{
			Error: convertToKeyError(err),
		}, nil
	}
	return &kvrpcpb.GetResponse{
		Value: val,
	}, nil
}

func (s *storeInstance) KvScan(ctx context.Context, req *kvrpcpb.ScanRequest) (*kvrpcpb.ScanResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.ScanResponse{RegionError: regionErr}, nil
	}
	endKey := reqCtx.endKey
	if len(req.GetEndKey()) > 0 && (len(endKey) == 0 || bytes.Compare(req.GetEndKey(), endKey) < 0) {
		endKey = req.GetEndKey()
	}
	pairs := reqCtx.store.Scan(req.GetStartKey(), endKey, int(req.GetLimit()), req.GetVersion(), reqCtx.isolationLevel)
	if req.GetKeyOnly() {
		return &kvrpcpb.ScanResponse{
			Pairs: convertToPbPairsKeyOnly(pairs),
		}, nil
	}
	return &kvrpcpb.ScanResponse{
		Pairs: convertToPbPairs(pairs),
	}, nil
}

func (s *storeInstance) KvPrewrite(ctx context.Context, req *kvrpcpb.PrewriteRequest) (*kvrpcpb.PrewriteResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.PrewriteResponse{RegionError: regionErr}, nil
	}
	for _, m := range req.Mutations {
		if !reqCtx.checkKeyInRegion(m.GetKey()) {
			panic("KvPrewrite: key not in region")
		}
	}
	errs := reqCtx.store.Prewrite(req.GetMutations(), req.GetPrimaryLock(), req.GetStartVersion(), req.GetLockTtl())
	return &kvrpcpb.PrewriteResponse{
		Errors: convertToKeyErrors(errs),
	}, nil
}

func (s *storeInstance) KvCommit(ctx context.Context, req *kvrpcpb.CommitRequest) (*kvrpcpb.CommitResponse, error) {
	switch s.rpcCommitResult {
	case "timeout":
		return nil, errors.New("timeout")
	case "notLeader":
		return &kvrpcpb.CommitResponse{RegionError: &errorpb.Error{NotLeader: &errorpb.NotLeader{}}}, nil
	case "keyError":
		return &kvrpcpb.CommitResponse{Error: &kvrpcpb.KeyError{}}, nil
	}
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.CommitResponse{RegionError: regionErr}, nil
	}
	for _, k := range req.Keys {
		if !reqCtx.checkKeyInRegion(k) {
			panic("KvCommit: key not in region")
		}
	}
	var resp kvrpcpb.CommitResponse
	err := reqCtx.store.Commit(req.GetKeys(), req.GetStartVersion(), req.GetCommitVersion())
	if err != nil {
		resp.Error = convertToKeyError(err)
	} else if s.rpcCommitTimeout {
		return nil, errUndetermined
	}
	return &resp, nil
}

func (s *storeInstance) KvImport(ctx context.Context, req *kvrpcpb.ImportRequest) (*kvrpcpb.ImportResponse, error) {
	panic("unimplemented")
}

func (s *storeInstance) KvCleanup(ctx context.Context, req *kvrpcpb.CleanupRequest) (*kvrpcpb.CleanupResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.CleanupResponse{RegionError: regionErr}, nil
	}
	if !reqCtx.checkKeyInRegion(req.GetKey()) {
		panic("KvCleanup: key not in region")
	}
	var resp kvrpcpb.CleanupResponse
	err := reqCtx.store.Cleanup(req.GetKey(), req.GetStartVersion())
	if err != nil {
		if commitTS, ok := errors.Cause(err).(ErrAlreadyCommitted); ok {
			resp.CommitVersion = uint64(commitTS)
		} else {
			resp.Error = convertToKeyError(err)
		}
	}
	return &resp, nil
}

func (s *storeInstance) KvBatchGet(ctx context.Context, req *kvrpcpb.BatchGetRequest) (*kvrpcpb.BatchGetResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.BatchGetResponse{RegionError: regionErr}, nil
	}
	for _, k := range req.GetKeys() {
		if !reqCtx.checkKeyInRegion(k) {
			panic("KvBatchGet: key not in region")
		}
	}
	pairs := reqCtx.store.BatchGet(req.GetKeys(), req.GetVersion(), reqCtx.isolationLevel)
	return &kvrpcpb.BatchGetResponse{
		Pairs: convertToPbPairs(pairs),
	}, nil
}

func (s *storeInstance) KvBatchRollback(ctx context.Context, req *kvrpcpb.BatchRollbackRequest) (*kvrpcpb.BatchRollbackResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.BatchRollbackResponse{RegionError: regionErr}, nil
	}
	err := reqCtx.store.Rollback(req.GetKeys(), req.GetStartVersion())
	if err != nil {
		return &kvrpcpb.BatchRollbackResponse{
			Error: convertToKeyError(err),
		}, nil
	}
	return &kvrpcpb.BatchRollbackResponse{}, nil
}

func (s *storeInstance) KvScanLock(ctx context.Context, req *kvrpcpb.ScanLockRequest) (*kvrpcpb.ScanLockResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.ScanLockResponse{RegionError: regionErr}, nil
	}
	startKey := MvccKey(reqCtx.startKey).Raw()
	endKey := MvccKey(reqCtx.endKey).Raw()
	locks, err := reqCtx.store.ScanLock(startKey, endKey, req.GetMaxVersion())
	if err != nil {
		return &kvrpcpb.ScanLockResponse{
			Error: convertToKeyError(err),
		}, nil
	}
	return &kvrpcpb.ScanLockResponse{
		Locks: locks,
	}, nil
}

func (s *storeInstance) KvResolveLock(ctx context.Context, req *kvrpcpb.ResolveLockRequest) (*kvrpcpb.ResolveLockResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.ResolveLockResponse{RegionError: regionErr}, nil
	}
	startKey := MvccKey(reqCtx.startKey).Raw()
	endKey := MvccKey(reqCtx.endKey).Raw()
	err := reqCtx.store.ResolveLock(startKey, endKey, req.GetStartVersion(), req.GetCommitVersion())
	if err != nil {
		return &kvrpcpb.ResolveLockResponse{
			Error: convertToKeyError(err),
		}, nil
	}
	return &kvrpcpb.ResolveLockResponse{}, nil
}

func (s *storeInstance) KvGC(ctx context.Context, req *kvrpcpb.GCRequest) (*kvrpcpb.GCResponse, error) {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.GCResponse{RegionError: regionErr}, nil
	}
	return &kvrpcpb.GCResponse{}, nil
}

func (s *storeInstance) KvDeleteRange(ctx context.Context, req *kvrpcpb.DeleteRangeRequest) (*kvrpcpb.DeleteRangeResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.DeleteRangeResponse{RegionError: regionErr}, nil
	}
	var resp kvrpcpb.DeleteRangeResponse
	err := reqCtx.store.DeleteRange(req.GetStartKey(), req.GetEndKey())
	if err != nil {
		resp.Error = err.Error()
	}
	return &resp, nil
}

// RawKV commands.
func (s *storeInstance) RawGet(ctx context.Context, req *kvrpcpb.RawGetRequest) (*kvrpcpb.RawGetResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawGetResponse{RegionError: regionErr}, nil
	}
	return &kvrpcpb.RawGetResponse{
		Value: reqCtx.store.RawGet(req.GetKey()),
	}, nil
}

func (s *storeInstance) RawBatchGet(ctx context.Context, req *kvrpcpb.RawBatchGetRequest) (*kvrpcpb.RawBatchGetResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawBatchGetResponse{RegionError: regionErr}, nil
	}
	pairs := reqCtx.store.RawBatchGet(req.GetKeys())
	return &kvrpcpb.RawBatchGetResponse{
		Pairs: convertToPbPairs(pairs),
	}, nil
}

func (s *storeInstance) RawPut(ctx context.Context, req *kvrpcpb.RawPutRequest) (*kvrpcpb.RawPutResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawPutResponse{RegionError: regionErr}, nil
	}
	reqCtx.store.RawPut(req.GetKey(), req.GetValue())
	return &kvrpcpb.RawPutResponse{}, nil
}

func (s *storeInstance) RawBatchPut(ctx context.Context, req *kvrpcpb.RawBatchPutRequest) (*kvrpcpb.RawBatchPutResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawBatchPutResponse{RegionError: regionErr}, nil
	}
	keys := make([][]byte, 0, len(req.GetPairs()))
	values := make([][]byte, 0, len(req.GetPairs()))
	for _, pair := range req.Pairs {
		keys = append(keys, pair.GetKey())
		values = append(values, pair.GetValue())
	}
	reqCtx.store.RawBatchPut(keys, values)
	return &kvrpcpb.RawBatchPutResponse{}, nil
}

func (s *storeInstance) RawDelete(ctx context.Context, req *kvrpcpb.RawDeleteRequest) (*kvrpcpb.RawDeleteResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawDeleteResponse{RegionError: regionErr}, nil
	}
	reqCtx.store.RawDelete(req.GetKey())
	return &kvrpcpb.RawDeleteResponse{}, nil
}

func (s *storeInstance) RawBatchDelete(ctx context.Context, req *kvrpcpb.RawBatchDeleteRequest) (*kvrpcpb.RawBatchDeleteResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawBatchDeleteResponse{RegionError: regionErr}, nil
	}
	reqCtx.store.RawBatchDelete(req.GetKeys())
	return &kvrpcpb.RawBatchDeleteResponse{}, nil
}

func (s *storeInstance) RawScan(ctx context.Context, req *kvrpcpb.RawScanRequest) (*kvrpcpb.RawScanResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawScanResponse{RegionError: regionErr}, nil
	}
	pairs := reqCtx.store.RawScan(req.GetStartKey(), req.GetEndKey(), int(req.GetLimit()))
	if req.GetKeyOnly() {
		return &kvrpcpb.RawScanResponse{
			Kvs: convertToPbPairsKeyOnly(pairs),
		}, nil
	}
	return &kvrpcpb.RawScanResponse{
		Kvs: convertToPbPairs(pairs),
	}, nil
}

func (s *storeInstance) RawDeleteRange(ctx context.Context, req *kvrpcpb.RawDeleteRangeRequest) (*kvrpcpb.RawDeleteRangeResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawDeleteRangeResponse{RegionError: regionErr}, nil
	}
	reqCtx.store.RawDeleteRange(req.GetStartKey(), req.GetEndKey())
	return &kvrpcpb.RawDeleteRangeResponse{}, nil
}

func (s *storeInstance) RawBatchScan(ctx context.Context, req *kvrpcpb.RawBatchScanRequest) (*kvrpcpb.RawBatchScanResponse, error) {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.RawBatchScanResponse{RegionError: regionErr}, nil
	}
	panic("unimplemented")
}

// Store commands (to the whole tikv but not a certain region)
func (s *storeInstance) UnsafeDestroyRange(ctx context.Context, req *kvrpcpb.UnsafeDestroyRangeRequest) (*kvrpcpb.UnsafeDestroyRangeResponse, error) {
	panic("unimplemented")
}

// SQL push down commands.
func (s *storeInstance) Coprocessor(ctx context.Context, req *coprocessor.Request) (*coprocessor.Response, error) {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &coprocessor.Response{RegionError: regionErr}, nil
	}
	return nil, nil
}

func (s *storeInstance) CoprocessorStream(req *coprocessor.Request, stream tikvpb.Tikv_CoprocessorStreamServer) error {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		stream.Send(&coprocessor.Response{RegionError: regionErr})
		return nil
	}
	return nil
}

// Raft commands (tikv <-> tikv).
func (s *storeInstance) Raft(tikvpb.Tikv_RaftServer) error {
	panic("unimplemented")
}

func (s *storeInstance) BatchRaft(tikvpb.Tikv_BatchRaftServer) error {
	panic("unimplemented")
}

func (s *storeInstance) Snapshot(tikvpb.Tikv_SnapshotServer) error {
	panic("unimplemented")
}

// Region commands.
func (s *storeInstance) SplitRegion(ctx context.Context, req *kvrpcpb.SplitRegionRequest) (*kvrpcpb.SplitRegionResponse, error) {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.SplitRegionResponse{RegionError: regionErr}, nil
	}
	key := NewMvccKey(req.GetSplitKey())
	region, _ := s.cluster.getRegionByKey(key)
	if bytes.Equal(region.GetStartKey(), key) {
		return &kvrpcpb.SplitRegionResponse{}, nil
	}
	/* TODO
	newRegionID, newPeerIDs := h.cluster.AllocID(), h.cluster.AllocIDs(len(region.Peers))
	h.cluster.SplitRaw(region.GetId(), newRegionID, key, newPeerIDs, newPeerIDs[0])
	*/
	return nil, nil
}

// transaction debugger commands.
func (s *storeInstance) MvccGetByKey(ctx context.Context, req *kvrpcpb.MvccGetByKeyRequest) (*kvrpcpb.MvccGetByKeyResponse, error) {
	reqCtx, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.MvccGetByKeyResponse{RegionError: regionErr}, nil
	}
	if !reqCtx.checkKeyInRegion(req.GetKey()) {
		panic("MvccGetByKey: key not in region")
	}
	return &kvrpcpb.MvccGetByKeyResponse{
		Error: "not implemented",
	}, nil
}

func (s *storeInstance) MvccGetByStartTs(ctx context.Context, req *kvrpcpb.MvccGetByStartTsRequest) (*kvrpcpb.MvccGetByStartTsResponse, error) {
	_, regionErr := s.checkRequest(req.GetContext(), req.Size())
	if regionErr != nil {
		return &kvrpcpb.MvccGetByStartTsResponse{RegionError: regionErr}, nil
	}
	return &kvrpcpb.MvccGetByStartTsResponse{
		Error: "not implemented",
	}, nil
}

func (s *storeInstance) handleBatchRequest(ctx context.Context, request *tikvpb.BatchCommandsRequest_Request) (resp *tikvpb.BatchCommandsResponse_Response, err error) {
	resp = &tikvpb.BatchCommandsResponse_Response{}
	if req := request.GetGet(); req != nil {
		r, e := s.KvGet(ctx, req)
		if e != nil {
			err = e
			return
		}
		resp.Cmd = &tikvpb.BatchCommandsResponse_Response_Get{
			Get: r,
		}
	} else if req := request.GetScan(); req != nil {
		r, e := s.KvScan(ctx, req)
		if e != nil {
			err = e
			return
		}
		resp.Cmd = &tikvpb.BatchCommandsResponse_Response_Scan{
			Scan: r,
		}
	}
	return
}

func (s *storeInstance) BatchCommands(stream tikvpb.Tikv_BatchCommandsServer) error {
	for {
		ctx := context.Background()
		requests, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.WithStack(err)
		}

		size := len(requests.Requests)
		responses := &tikvpb.BatchCommandsResponse{
			Responses:          make([]*tikvpb.BatchCommandsResponse_Response, size),
			RequestIds:         make([]uint64, size),
			TransportLayerLoad: 0,
		}
		for idx, request := range requests.Requests {
			responses.RequestIds[idx] = requests.RequestIds[idx]
			r, e := s.handleBatchRequest(ctx, request)
			if e != nil {
				err = e
				break
			}
			responses.Responses[idx] = r
		}
		if e := stream.Send(responses); e != nil {
			return errors.WithStack(e)
		}
		if err != nil {
			return errors.WithStack(err)
		}
	}
}

func (s *storeInstance) getFailPoints() map[string]interface{} {
	return map[string]interface{}{
		"rpc-commit-result":  s.rpcCommitResult,
		"rpc-commit-timeout": s.rpcCommitTimeout,
	}
}

func (s *storeInstance) updateFailPoint(failPoint, value string) (interface{}, error) {
	switch failPoint {
	case serverBusyFailPoint:
		s.ctx = failpoint.WithHook(s.ctx, func(ctx context.Context, fpname string) bool {
			return fpname == "github.com/tikv/mock-tikv/server/" + serverBusyFailPoint
		})
		if err := failpoint.Enable("github.com/tikv/mock-tikv/server/" + serverBusyFailPoint, value); err != nil {
			return nil, err
		}
		return nil, nil
	case "rpc-commit-result":
		switch value {
		case "timeout":
			fallthrough
		case "notLeader":
			fallthrough
		case "keyError":
			fallthrough
		case "":
			s.rpcCommitResult = value
		default:
			return nil, errInvalidFailPointValue
		}
		return s.rpcCommitResult, nil
	case "rpc-commit-timeout":
		if value == "" {
			s.rpcCommitTimeout = false
		} else {
			commitTimeout, err := strconv.ParseBool(value)
			if err != nil {
				return nil, err
			}
			s.rpcCommitTimeout = commitTimeout
		}
		return s.rpcCommitTimeout, nil
	}
	return nil, errFailPointNotFound
}
