package libp2pgrpc

import (
	"errors"
	"net"
	"testing"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
)

func TestErrorRasiedBeforeServe(t *testing.T) {
	origin_gostream_listen_func := _gostream_Listen
	srv := Server{}
	// mock function
	_gostream_Listen = func(host.Host, protocol.ID) (net.Listener, error) {
		return nil, errors.New("mock error before serve()")
	}
	assert.Equal(t, "mock error before serve()", srv.Serve().Error())
	_gostream_Listen = origin_gostream_listen_func
}
