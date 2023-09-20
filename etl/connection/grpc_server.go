package connection

import (
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/insolar/block-explorer/configuration"
	"google.golang.org/grpc"
)

// NewGRPCServer configures the gRPC server with metrics
func NewGRPCServer(cfg configuration.Exporter, grpcMetrics *grpc_prometheus.ServerMetrics) (*grpc.Server, error) {
	return grpc.NewServer(
		grpc.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpcMetrics.StreamServerInterceptor()),
	), nil
}
