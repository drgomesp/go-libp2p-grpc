package libp2pgrpc

import (
	"context"
	"log"
	"net"

	gostream "github.com/libp2p/go-libp2p-gostream"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/grpc"
)

func (c *Client) GetDialOption(ctx context.Context) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, peerIdStr string) (net.Conn, error) {
		peerID, err := peer.Decode(peerIdStr)
		if err != nil {
			log.Fatal(err)

			return nil, err
		}

		conn, err := gostream.Dial(ctx, c.host, peerID, ProtocolID)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		return conn, nil
	})
}

func (c *Client) Dial(
	ctx context.Context,
	peerID peer.ID,
	dialOpts ...grpc.DialOption,
) (*grpc.ClientConn, error) {
	dialOpsPrepended := append([]grpc.DialOption{c.GetDialOption(ctx)}, dialOpts...)
	return grpc.DialContext(ctx, peerID.String(), dialOpsPrepended...)
}
