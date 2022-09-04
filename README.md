# go-libp2p-grpc

[![build](https://github.com/drgomesp/go-libp2p-grpc/actions/workflows/go-test.yml/badge.svg?style=squared)](https://github.com/drgomesp/go-libp2p-grpc/actions)
[![codecov](https://codecov.io/gh/drgomesp/go-libp2p-grpc/branch/main/graph/badge.svg?token=BRMFJRJV2X)](https://codecov.io/gh/drgomesp/go-libp2p-grpc)

> ⚙ GRPC/Protobuf on Libp2p.

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Install

```bash
go get github.com/drgomesp/go-libp2p-grpc
```

## Usage

Given an RPC service:

```proto
service EchoService {
  // Echo asks a node to respond with a message.
  rpc Echo(EchoRequest) returns (EchoReply) {}
}
```

```go
type EchoService struct {}

func (EchoService) Echo(context.Context, *EchoRequest) (*EchoReply, error) {
	...
}
```

And a libp2p host to act as the server:

```go
ma, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")

serverHost, err := libp2p.New(libp2p.ListenAddrs(ma))
if err != nil {
    log.Fatal(err)
}
defer serverHost.Close()

srv, err := libp2pgrpc.NewGrpcServer(ctx, serverHost)
if err != nil {
    log.Fatal(err)
}
```

Register the GRPC service to the host server:
```go
pb.RegisterEchoServiceServer(srv, &EchoService{})
```

A libp2p host to act as the client:
```go
ma, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")

clientHost, err := libp2p.New(libp2p.ListenAddrs(ma))
if err != nil {
    log.Fatal(err)
}
```

Dial the server and initiate the request:

```go
client := libp2pgrpc.NewClient(cliHost, libp2pgrpc.ProtocolID, libp2pgrpc.WithServer(srv))
conn, err := client.Dial(ctx, srvHost.ID(), grpc.WithTransportCredentials(insecure.NewCredentials()))
assert.NoError(t, err)
defer conn.Close()

c := pb.NewEchoServiceClient(conn)
res, err := c.Echo(ctx, &pb.EchoRequest{Message: "here's your response"})
```

## Contributing

PRs accepted.

## License

MIT © [Daniel Ribeiro](https://github.com/drgomesp)