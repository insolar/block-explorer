package exporter

import (
	"context"
	"net"
	"sync"

	"github.com/insolar/block-explorer/instrumentation/belogger"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Server implements introspection gRPC server.
type Server struct {
	listen string

	grpcServer *grpc.Server

	cancel         context.CancelFunc
	hasStarted     bool
	startStopMutex *sync.Mutex
}

func NewServer(listen string, grpcServer *grpc.Server) *Server {
	return &Server{
		listen:         listen,
		grpcServer:     grpcServer,
		hasStarted:     false,
		startStopMutex: &sync.Mutex{},
	}
}

func (s *Server) Start(ctx context.Context) error {
	if s.grpcServer == nil {
		return errors.New("gRPC server is required")
	}

	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()
	if s.hasStarted {
		return nil
	}

	l, err := net.Listen("tcp", s.listen)
	if err != nil {
		return errors.Wrapf(err, "failed to start gPRC server on %s", s.listen)
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go s.run(ctx, l)
	belogger.FromContext(ctx).
		Infof("started gPRC server on %s\n", l.Addr())

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.startStopMutex.Lock()
	defer s.startStopMutex.Unlock()
	if !s.hasStarted {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	belogger.FromContext(ctx).
		Warn("stop called for not started introspection server")
	return nil
}

func (s *Server) run(ctx context.Context, l net.Listener) {
	go func(ctx context.Context, grpcServer *grpc.Server) {
		err := grpcServer.Serve(l)
		if err != nil {
			belogger.FromContext(ctx).Error("gRPC server stopped gracefully: ", err)
		}
	}(ctx, s.grpcServer)
}
