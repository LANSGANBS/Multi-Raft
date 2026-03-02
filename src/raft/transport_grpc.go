package raft

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/LANSGANBS/Multi-Raft/src/raft/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcTransport struct {
	mu        sync.RWMutex
	me        int
	peers     map[int]string
	listenAddr string
	handler   RPCHandler
	
	server   *grpc.Server
	clients  map[int]pb.RaftServiceClient
	conns    map[int]*grpc.ClientConn
	
	closed   bool
	closeCh  chan struct{}
}

func NewGrpcTransport(config *TransportConfig) *GrpcTransport {
	t := &GrpcTransport{
		me:         config.PeerID,
		peers:      make(map[int]string),
		listenAddr: config.ListenAddr,
		clients:    make(map[int]pb.RaftServiceClient),
		conns:      make(map[int]*grpc.ClientConn),
		closeCh:    make(chan struct{}),
	}
	
	for i, addr := range config.Peers {
		t.peers[i] = addr
	}
	
	return t
}

func (t *GrpcTransport) GetPeerCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.peers)
}

func (t *GrpcTransport) GetPeerID() int {
	return t.me
}

func (t *GrpcTransport) GetEndpoint(peerID int) RPCEndpoint {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if client, ok := t.clients[peerID]; ok {
		return &grpcEndpoint{client: client, timeout: 5 * time.Second}
	}
	return nil
}

func (t *GrpcTransport) RegisterHandler(handler RPCHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

func (t *GrpcTransport) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	lis, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		return err
	}
	
	t.server = grpc.NewServer()
	pb.RegisterRaftServiceServer(t.server, &raftServiceServer{handler: t})
	
	go t.server.Serve(lis)
	
	for id, addr := range t.peers {
		if id == t.me {
			continue
		}
		go t.connectToPeer(id, addr)
	}
	
	return nil
}

func (t *GrpcTransport) connectToPeer(id int, addr string) {
	for {
		t.mu.RLock()
		closed := t.closed
		t.mu.RUnlock()
		
		if closed {
			return
		}
		
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		
		t.mu.Lock()
		if t.closed {
			conn.Close()
			t.mu.Unlock()
			return
		}
		if oldConn, ok := t.conns[id]; ok {
			oldConn.Close()
		}
		t.conns[id] = conn
		t.clients[id] = pb.NewRaftServiceClient(conn)
		t.mu.Unlock()
		
		return
	}
}

func (t *GrpcTransport) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return
	}
	
	t.closed = true
	close(t.closeCh)
	
	if t.server != nil {
		t.server.GracefulStop()
	}
	
	for _, conn := range t.conns {
		conn.Close()
	}
}

func (t *GrpcTransport) getHandler() RPCHandler {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.handler
}

type grpcEndpoint struct {
	client  pb.RaftServiceClient
	timeout time.Duration
}

func (e *grpcEndpoint) SendRequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	resp, err := e.client.RequestVote(ctx, &pb.RequestVoteRequest{
		Term:         int32(req.Term),
		CandidateId:  int32(req.CandidateId),
		LastLogIndex: int32(req.LastLogIndex),
		LastLogTerm:  int32(req.LastLogTerm),
	})
	
	if err != nil {
		return nil, err
	}
	
	return &RequestVoteResponse{
		Term:        int(resp.Term),
		VoteGranted: resp.VoteGranted,
	}, nil
}

func (e *grpcEndpoint) SendAppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	entries := make([]*pb.LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = &pb.LogEntry{
			Term:         int32(e.Term),
			CommandValid: e.CommandValid,
			Command:      marshalCommand(e.Command),
		}
	}
	
	resp, err := e.client.AppendEntries(ctx, &pb.AppendEntriesRequest{
		Term:         int32(req.Term),
		LeaderId:     int32(req.LeaderId),
		PrevLogIndex: int32(req.PrevLogIndex),
		PrevLogTerm:  int32(req.PrevLogTerm),
		Entries:      entries,
		LeaderCommit: int32(req.LeaderCommit),
	})
	
	if err != nil {
		return nil, err
	}
	
	return &AppendEntriesResponse{
		Term:         int(resp.Term),
		Success:      resp.Success,
		ConflictIdx:  int(resp.ConflictIdx),
		ConflictTerm: int(resp.ConflictTerm),
	}, nil
}

func (e *grpcEndpoint) SendInstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	resp, err := e.client.InstallSnapshot(ctx, &pb.InstallSnapshotRequest{
		Term:              int32(req.Term),
		LeaderId:          int32(req.LeaderId),
		LastIncludedIndex: int32(req.LastIncludedIndex),
		LastIncludedTerm:  int32(req.LastIncludedTerm),
		Snapshot:          req.Snapshot,
	})
	
	if err != nil {
		return nil, err
	}
	
	return &InstallSnapshotResponse{
		Term: int(resp.Term),
	}, nil
}

type raftServiceServer struct {
	pb.UnimplementedRaftServiceServer
	handler *GrpcTransport
}

func (s *raftServiceServer) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	handler := s.handler.getHandler()
	if handler == nil {
		return nil, context.Canceled
	}
	
	resp := handler.HandleRequestVote(&RequestVoteRequest{
		Term:         int(req.Term),
		CandidateId:  int(req.CandidateId),
		LastLogIndex: int(req.LastLogIndex),
		LastLogTerm:  int(req.LastLogTerm),
	})
	
	return &pb.RequestVoteResponse{
		Term:        int32(resp.Term),
		VoteGranted: resp.VoteGranted,
	}, nil
}

func (s *raftServiceServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	handler := s.handler.getHandler()
	if handler == nil {
		return nil, context.Canceled
	}
	
	entries := make([]LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = LogEntry{
			Term:         int(e.Term),
			CommandValid: e.CommandValid,
			Command:      unmarshalCommand(e.Command),
		}
	}
	
	resp := handler.HandleAppendEntries(&AppendEntriesRequest{
		Term:         int(req.Term),
		LeaderId:     int(req.LeaderId),
		PrevLogIndex: int(req.PrevLogIndex),
		PrevLogTerm:  int(req.PrevLogTerm),
		Entries:      entries,
		LeaderCommit: int(req.LeaderCommit),
	})
	
	return &pb.AppendEntriesResponse{
		Term:         int32(resp.Term),
		Success:      resp.Success,
		ConflictIdx:  int32(resp.ConflictIdx),
		ConflictTerm: int32(resp.ConflictTerm),
	}, nil
}

func (s *raftServiceServer) InstallSnapshot(ctx context.Context, req *pb.InstallSnapshotRequest) (*pb.InstallSnapshotResponse, error) {
	handler := s.handler.getHandler()
	if handler == nil {
		return nil, context.Canceled
	}
	
	resp := handler.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              int(req.Term),
		LeaderId:          int(req.LeaderId),
		LastIncludedIndex: int(req.LastIncludedIndex),
		LastIncludedTerm:  int(req.LastIncludedTerm),
		Snapshot:          req.Snapshot,
	})
	
	return &pb.InstallSnapshotResponse{
		Term: int32(resp.Term),
	}, nil
}

func marshalCommand(cmd interface{}) []byte {
	if cmd == nil {
		return nil
	}
	switch v := cmd.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	default:
		return nil
	}
}

func unmarshalCommand(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	return data
}
