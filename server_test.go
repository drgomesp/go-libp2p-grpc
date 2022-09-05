package libp2pgrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	proto "github.com/drgomesp/go-libp2p-grpc/proto/v1"
)

type NodeInfoService struct {
	proto.UnimplementedNodeServiceServer

	host host.Host
}

// Info returns information about the node service's underlying host.
func (s *NodeInfoService) Info(context.Context, *proto.InfoRequest) (*proto.InfoResponse, error) {
	return &proto.InfoResponse{
		PeerId:    s.host.ID().String(),
		Addresses: addresses(s),
		Protocols: s.host.Mux().Protocols(),
	}, nil
}

func addresses(s *NodeInfoService) []string {
	res := make([]string, 0)

	for _, addr := range s.host.Addrs() {
		res = append(res, addr.String())
	}

	return res
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
	svc := &NodeInfoService{host: srvHost}
	proto.RegisterNodeServiceServer(srv, svc)

	client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	c := proto.NewNodeServiceClient(conn)
	res, err := c.Info(ctx, &proto.InfoRequest{})

	assert.NoError(t, err)
	assert.Equal(t, srvHost.ID().String(), res.PeerId)
	assert.Equal(t, addresses(svc), res.Addresses)
	assert.Equal(t, srvHost.Mux().Protocols(), res.Protocols)
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
	svc := &NodeInfoService{host: srvHost}
	proto.RegisterNodeServiceServer(srv, svc)

	client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	mux := runtime.NewServeMux()
	err = proto.RegisterNodeServiceHandler(ctx, mux, conn)
	assert.NoError(t, err)

	c := proto.NewNodeServiceClient(conn)
	res, err := c.Info(ctx, &proto.InfoRequest{})

	assert.NoError(t, err)
	assert.Equal(t, srvHost.ID().String(), res.PeerId)
	assert.Equal(t, addresses(svc), res.Addresses)
	assert.Equal(t, srvHost.Mux().Protocols(), res.Protocols)

	go func() {
		http.ListenAndServe(":4000", mux)
	}()
	httpClient := &http.Client{}
	response, err := httpClient.Get(
		"http://localhost:4000/v1/node/info",
	)
	assert.NoError(t, err)

	expected := &proto.InfoResponse{
		PeerId:    srvHost.ID().String(),
		Addresses: addresses(svc),
		Protocols: srvHost.Mux().Protocols(),
	}
	expectedData, err := json.Marshal(expected)
	var buf bytes.Buffer
	assert.NoError(t, json.Compact(&buf, expectedData))
	assert.NoError(t, err)

	data, err := io.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(expectedData), string(data))
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
	proto.RegisterNodeServiceServer(srv, &NodeInfoService{host: srvHost})

	client := libp2pgrpc.NewClient(cliHost, "/bad/proto")
	conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	c := proto.NewNodeServiceClient(conn)
	res, err := c.Info(ctx, &proto.InfoRequest{})

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Equal(t, "rpc error: code = Unavailable desc = connection error: desc = \"transport: Error while dialing protocol not supported\"", err.Error())
}
