package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	proto "github.com/drgomesp/go-libp2p-grpc/proto/v1"
)

type NodeService struct {
	proto.UnimplementedNodeServiceServer

	host host.Host
}

// Info returns information about the node service's underlying host.
func (s *NodeService) Info(context.Context, *proto.InfoRequest) (*proto.InfoResponse, error) {
	return &proto.InfoResponse{
		PeerId: s.host.ID().String(),
		Addresses: func() []string {
			res := make([]string, 0)

			for _, addr := range s.host.Addrs() {
				res = append(res, addr.String())
			}

			return res
		}(),
		Protocols: s.host.Mux().Protocols(),
	}, nil
}

func main() {
	ctx := context.Background()

	m1, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	m2, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")

	serverHost, err := libp2p.New(libp2p.ListenAddrs(m1))
	if err != nil {
		log.Fatal(err)
	}

	clientHost, err := libp2p.New(libp2p.ListenAddrs(m2))
	if err != nil {
		log.Fatal(err)
	}

	defer clientHost.Close()
	defer serverHost.Close()

	serverHost.Peerstore().AddAddrs(clientHost.ID(), clientHost.Addrs(), peerstore.PermanentAddrTTL)
	clientHost.Peerstore().AddAddrs(serverHost.ID(), serverHost.Addrs(), peerstore.PermanentAddrTTL)

	srv, err := libp2pgrpc.NewGrpcServer(ctx, serverHost)
	if err != nil {
		log.Fatal(err)
	}

	proto.RegisterNodeServiceServer(srv, &NodeService{host: serverHost})
	client := libp2pgrpc.NewClient(clientHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, serverHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	//c := proto.NewNodeServiceClient(conn)
	//res, err := c.Info(ctx, &proto.InfoRequest{})
	//if err != nil {
	//	log.Fatal(err)
	//}

	mux := runtime.NewServeMux()
	err = proto.RegisterNodeServiceHandler(ctx, mux, conn)
	if err != nil {
		log.Fatal(err)
	}

	addr := "localhost:4000"
	log.Printf("Visit http://%s/v1/node/info\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
