package main

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	"github.com/drgomesp/go-libp2p-grpc/examples/echo/proto"
)

type EchoService struct {
	proto.UnimplementedEchoServiceServer
}

func (s *EchoService) Echo(context.Context, *proto.EchoRequest) (*proto.EchoReply, error) {
	return &proto.EchoReply{
		Message: "heyo!",
		PeerId:  "123",
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

	proto.RegisterEchoServiceServer(srv, &EchoService{})
	client := libp2pgrpc.NewClient(clientHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, serverHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	c := proto.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &proto.EchoRequest{Message: "give me something"})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(res.Message)
}
