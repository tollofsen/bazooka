// Package server exposes the backend over gRPC.
package server

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tollofsen/bazooka/backend/internal/router"
	"github.com/tollofsen/bazooka/backend/internal/session"
	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

// Server implements pb.BazookaServer.
type Server struct {
	pb.UnimplementedBazookaServer
	sessions *session.Manager
	router   *router.Router
}

// New creates a gRPC service backed by the given manager and router.
func New(m *session.Manager, r *router.Router) *Server {
	return &Server{sessions: m, router: r}
}

func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	// The dial must outlive the RPC only through the session, so ctx is used
	// just for the connect timeout/cancellation.
	sess, err := s.sessions.Connect(ctx, req.SessionId, req.Host, int(req.Port))
	switch {
	case errors.Is(err, session.ErrExists):
		return nil, status.Error(codes.AlreadyExists, err.Error())
	case err != nil:
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	return &pb.ConnectResponse{SessionId: sess.ID}, nil
}

func (s *Server) Disconnect(_ context.Context, req *pb.DisconnectRequest) (*pb.DisconnectResponse, error) {
	if err := s.sessions.Disconnect(req.SessionId); err != nil {
		return nil, toStatus(err)
	}
	return &pb.DisconnectResponse{}, nil
}

func (s *Server) SendCommand(_ context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	sess, err := s.sessions.Get(req.SessionId)
	if err != nil {
		return nil, toStatus(err)
	}
	if err := sess.Send(req.Command, req.Source); err != nil {
		return nil, toStatus(err)
	}
	return &pb.SendCommandResponse{}, nil
}

func (s *Server) StreamOutput(req *pb.StreamOutputRequest, stream pb.Bazooka_StreamOutputServer) error {
	sess, err := s.sessions.Get(req.SessionId)
	if err != nil {
		return toStatus(err)
	}
	// Subscribe before replaying so no live output is missed.
	ch, cancel := s.router.Subscribe(req.SessionId)
	defer cancel()
	for _, o := range sess.Scrollback(int(req.ReplayLines)) {
		if err := stream.Send(o); err != nil {
			return err
		}
	}
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case o := <-ch:
			if err := stream.Send(o); err != nil {
				return err
			}
		}
	}
}

func (s *Server) ListSessions(context.Context, *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	return &pb.ListSessionsResponse{Sessions: s.sessions.List()}, nil
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, session.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, session.ErrClosed):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
