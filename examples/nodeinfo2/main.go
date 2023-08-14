package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	mrand "math/rand"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	libp2pgrpc "github.com/drgomesp/go-libp2p-grpc"
	proto "github.com/drgomesp/go-libp2p-grpc/proto/v1"
	golog "github.com/ipfs/go-log/v2"
)

var log = golog.Logger("demo_main")

type NodeService struct {
	proto.UnimplementedNodeServiceServer

	host host.Host
}

// Info returns information about the node service's underlying host.
func (s *NodeService) Info(_ context.Context, _ *proto.NodeInfoRequest) (*proto.NodeInfoResponse, error) {
	peers := make([]string, 0)
	for _, peer := range s.host.Peerstore().Peers() {
		peers = append(peers, peer.ShortString())
	}
	sort.Strings(peers)
	log.Debugw("handle node info request")

	return &proto.NodeInfoResponse{
		Id: s.host.ID().String(),
		Addresses: func() []string {
			res := make([]string, 0)

			for _, addr := range s.host.Addrs() {
				res = append(res, addr.String())
			}

			return res
		}(),
		Protocols: protocol.ConvertToStrings(s.host.Mux().Protocols()),
		Peers:     peers,
	}, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(golog.LevelDebug)

	// Parse options from the command line
	listenF := flag.Int("l", 10000, "wait for incoming connections")
	targetF := flag.String("d", "", "target peer to dial")
	seedF := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	var r io.Reader
	if *seedF == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(*seedF))
	}
	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	check(err)

	if *targetF == "" {
		// for server
		m4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", *listenF))
		check(err)
		ha, err := libp2p.New(libp2p.Identity(priv), libp2p.ListenAddrs(m4), libp2p.DefaultSecurity)
		check(err)
		defer ha.Close()

		fullAddr := getHostAddress(ha)
		log.Debugf("I am %s", fullAddr)
		srv, err := libp2pgrpc.NewGrpcServer(ctx, ha)
		check(err)
		proto.RegisterNodeServiceServer(srv, &NodeService{host: ha})
		log.Infow("gRPC server is ready")
		go srv.Serve()
		// Run until canceled.
		<-ctx.Done()

	} else {
		// for client
		ha, err := libp2p.New(libp2p.Identity(priv), libp2p.DefaultSecurity)
		check(err)
		defer ha.Close()

		// Turn the targetPeer into a multiaddr.
		maddr, err := multiaddr.NewMultiaddr(*targetF)
		check(err)
		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		check(err)

		log.Debugw("Will get remote node info", "target", *targetF)
		// We have a peer ID and a targetAddr, so we add it to the peerstore
		// so LibP2P knows how to contact it
		ha.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

		// ha will act as the grpc client here, dialing the h2 server
		client := libp2pgrpc.NewClient(ha, libp2pgrpc.ProtocolID)
		opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())}
		conn, err := client.Dial(ctx, info.ID, opts...)
		check(err)

		nsc := proto.NewNodeServiceClient(conn)
		for i := 0; i < 3; i++ {
			req := proto.NodeInfoRequest{}
			res, err := nsc.Info(ctx, &req)
			if err != nil {
				log.Fatalf("error while calling NodeService RPC: %v", err)
			}
			log.Debugw("Response from remote", "i", i, "nodeInfo", res)
			time.Sleep(2 * time.Second)
		}
	}
	log.Debugw("exit")
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getHostAddress(ha host.Host) string {
	// Build host multiaddress
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", ha.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := ha.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}
