package libp2pgrpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	pb "github.com/drgomesp/go-libp2p-grpc/pb/examples/echo"
)

type GreeterService struct {
	pb.UnimplementedEchoServiceServer
}

func (s *GreeterService) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoReply, error) {
	return &pb.EchoReply{Message: fmt.Sprintf("%s ja pierdole", req.GetMessage())}, nil
}

func newHost(t *testing.T, listen multiaddr.Multiaddr) host.Host {
	h, err := libp2p.New(
		libp2p.ListenAddrs(listen),
	)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestGrpc(t *testing.T) {
	ctx := context.Background()

	m1, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	m2, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")

	srvHost := newHost(t, m1)
	defer srvHost.Close()

	cliHost := newHost(t, m2)
	defer cliHost.Close()

	srvHost.Peerstore().AddAddrs(cliHost.ID(), cliHost.Addrs(), peerstore.PermanentAddrTTL)
	cliHost.Peerstore().AddAddrs(srvHost.ID(), srvHost.Addrs(), peerstore.PermanentAddrTTL)

	srv, err := libp2pgrpc.NewGrpcServer(ctx, srvHost)
	assert.NoError(t, err)
	pb.RegisterEchoServiceServer(srv, &GreeterService{})

	client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	c := pb.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &pb.EchoRequest{Message: "kurwa mać"})

	assert.NoError(t, err)
	assert.Equal(t, "kurwa mać ja pierdole", res.Message)
}

func TestGrpcDialBadProtocol(t *testing.T) {
	ctx := context.Background()

	m1, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	m2, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")

	srvHost := newHost(t, m1)
	defer srvHost.Close()

	cliHost := newHost(t, m2)
	defer cliHost.Close()

	srvHost.Peerstore().AddAddrs(cliHost.ID(), cliHost.Addrs(), peerstore.PermanentAddrTTL)
	cliHost.Peerstore().AddAddrs(srvHost.ID(), srvHost.Addrs(), peerstore.PermanentAddrTTL)

	srv, err := libp2pgrpc.NewGrpcServer(ctx, srvHost)
	assert.NoError(t, err)
	pb.RegisterEchoServiceServer(srv, &GreeterService{})

	client := libp2pgrpc.NewClient(cliHost, "bad protocol", libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	c := pb.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &pb.EchoRequest{Message: "kurwa mać"})

	assert.Error(t, err)
	assert.Equal(t, errors.New("protocol not supported"), res.Message)
}