package libp2pgrpc

import (
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// ClientOption allows for functional setting of options on a Client.
type ClientOption func(*Client)

func WithServer(s *Server) ClientOption {
	return func(c *Client) {
		c.server = s
	}
}

type Client struct {
	host     host.Host
	protocol protocol.ID
	server   *Server
}

func NewClient(h host.Host, p protocol.ID, opts ...ClientOption) *Client {
	c := &Client{
		host:     h,
		protocol: p,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
