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

	h1, err := libp2p.New(libp2p.ListenAddrs(m1))
	check(err)

	h2, err := libp2p.New(libp2p.ListenAddrs(m2))
	check(err)

	defer h2.Close()
	defer h1.Close()

	h1.Peerstore().AddAddrs(h2.ID(), h2.Addrs(), peerstore.PermanentAddrTTL)
	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), peerstore.PermanentAddrTTL)

	ch := make(chan bool, 1)

	go func() {
		// initialize h1 as grpc server
		srv, err := libp2pgrpc.NewGrpcServer(ctx, h1)
		check(err)
		proto.RegisterNodeServiceServer(srv, &NodeService{host: h1})

		// h2 will act as the grpc client here, dialing the h1 server
		client := libp2pgrpc.NewClient(h2, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
		conn, err := client.Dial(ctx, h1.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		check(err)

		mux := runtime.NewServeMux()
		err = proto.RegisterNodeServiceHandler(ctx, mux, conn)
		check(err)

		addrStr := ":4000"
		log.Printf("node=%s url=http://localhost%s/v1/node/info\n", h1.ID().String(), addrStr)
		log.Fatal(http.ListenAndServe(addrStr, mux))
	}()

	go func() {
		// initialize h1 as grpc server
		srv, err := libp2pgrpc.NewGrpcServer(ctx, h2)
		check(err)
		proto.RegisterNodeServiceServer(srv, &NodeService{host: h2})

		// h1 will act as the grpc client here, dialing the h2 server
		client := libp2pgrpc.NewClient(h1, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
		conn, err := client.Dial(ctx, h2.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		check(err)

		mux := runtime.NewServeMux()
		err = proto.RegisterNodeServiceHandler(ctx, mux, conn)
		check(err)

		addrStr := ":4001"
		log.Printf("node=%s url=http://localhost%s/v1/node/info\n", h2.ID().String(), addrStr)
		log.Fatal(http.ListenAndServe(addrStr, mux))
	}()

	<-ch
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
