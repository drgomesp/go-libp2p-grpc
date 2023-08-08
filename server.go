package libp2pgrpc

import (
	"context"

	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/libp2p/go-libp2p/core/host"
	"google.golang.org/grpc"
)

var _ grpc.ServiceRegistrar = &Server{}

type Server struct {
	host host.Host
	grpc *grpc.Server
	ctx  context.Context
}

// NewGrpcServer creates a Server object with the given LibP2P host
// and protocol.
func NewGrpcServer(ctx context.Context, h host.Host, opts ...grpc.ServerOption) (*Server, error) {
	grpcServer := grpc.NewServer(opts...)
	srv := &Server{
		host: h,
		ctx:  ctx,
		grpc: grpcServer,
	}
	return srv, nil
}

func (s *Server) Serve() error {
	listener, err := gostream.Listen(s.host, ProtocolID)
	if err != nil {
		return err
	}
	return s.grpc.Serve(listener)
}

func (s *Server) RegisterService(serviceDesc *grpc.ServiceDesc, srv interface{}) {
	s.grpc.RegisterService(serviceDesc, srv)
}
