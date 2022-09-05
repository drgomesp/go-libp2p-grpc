package libp2pgrpc_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	"github.com/drgomesp/go-libp2p-grpc/examples/echo/proto"
)

type GreeterService struct {
	proto.UnimplementedEchoServiceServer
}

func (s *GreeterService) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoReply, error) {
	return &proto.EchoReply{Message: fmt.Sprintf("%s comes from here", req.GetMessage())}, nil
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

	srv, err := libp2pgrpc.NewGrpcServer(ctx, srvHost, grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    time.Second,
		Timeout: 100 * time.Millisecond,
	}))

	assert.NoError(t, err)
	proto.RegisterEchoServiceServer(srv, &GreeterService{})

	client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	c := proto.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &proto.EchoRequest{Message: "some message"})

	assert.NoError(t, err)
	assert.Equal(t, "some message comes from here", res.Message)
}

func TestGrpcGateway(t *testing.T) {
	ctx := context.Background()

	m1, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	m2, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")

	srvHost := newHost(t, m1)
	defer srvHost.Close()

	cliHost := newHost(t, m2)
	defer cliHost.Close()

	srvHost.Peerstore().AddAddrs(cliHost.ID(), cliHost.Addrs(), peerstore.PermanentAddrTTL)
	cliHost.Peerstore().AddAddrs(srvHost.ID(), srvHost.Addrs(), peerstore.PermanentAddrTTL)

	srv, err := libp2pgrpc.NewGrpcServer(ctx, srvHost, grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    time.Second,
		Timeout: 3 * time.Second,
	}))

	assert.NoError(t, err)
	proto.RegisterEchoServiceServer(srv, &GreeterService{})

	client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	mux := runtime.NewServeMux()
	err = proto.RegisterEchoServiceHandler(ctx, mux, conn)
	assert.NoError(t, err)

	c := proto.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &proto.EchoRequest{Message: "some message"})

	assert.NoError(t, err)
	assert.Equal(t, "some message comes from here", res.Message)

	go func() {
		http.ListenAndServe(":4000", mux)
	}()

	httpClient := &http.Client{}
	response, err := httpClient.Get(
		fmt.Sprintf("http://localhost:4000/v1/example/echo"),
	)
	assert.NoError(t, err)

	data, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"message":" comes from here"}`, string(data))
}

func TestGrpcBadProtocol(t *testing.T) {
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
	proto.RegisterEchoServiceServer(srv, &GreeterService{})

	client := libp2pgrpc.NewClient(cliHost, "/bad/proto")
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	c := proto.NewEchoServiceClient(conn)
	res, err := c.Echo(ctx, &proto.EchoRequest{Message: "some message"})

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Equal(t, "rpc error: code = Unavailable desc = connection error: desc = \"transport: Error while dialing protocol not supported\"", err.Error())
}
