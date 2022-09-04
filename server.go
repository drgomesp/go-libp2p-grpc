package libp2pgrpc

import (
	"context"
	"log"

	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/libp2p/go-libp2p/core/host"
	"google.golang.org/grpc"
)

// ServerOption allows for functional setting of options on a Server.
type ServerOption func(*Server)

var _ grpc.ServiceRegistrar = &Server{}

type Server struct {
	host host.Host
	grpc *grpc.Server
	ctx  context.Context
}

// NewGrpcServer creates a Server object with the given LibP2P host
// and protocol.
func NewGrpcServer(ctx context.Context, h host.Host, opts ...ServerOption) *Server {
	grpcServer := grpc.NewServer()

	srv := &Server{
		host: h,
		ctx:  ctx,
		grpc: grpcServer,
	}

	for _, opt := range opts {
		opt(srv)
	}

	listener, err := gostream.Listen(srv.host, ProtocolID)
	if err != nil {
		log.Fatal(err)
	}

	go srv.grpc.Serve(listener)

	return srv
}

func (s *Server) GRPC() grpc.ServiceRegistrar {
	return s.grpc
}

func (s *Server) Register(svc *grpc.ServiceDesc) error {
	s.grpc.RegisterService(svc, s.grpc)

	return nil
}

func (s *Server) RegisterService(serviceDesc *grpc.ServiceDesc, srv interface{}) {
	s.grpc.RegisterService(serviceDesc, srv)
}
