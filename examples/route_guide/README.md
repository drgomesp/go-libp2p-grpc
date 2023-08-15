# Description
The route guide server and client demonstrate how to use grpc go libraries to
perform unary, client streaming, server streaming and full duplex RPCs.

Please refer to [gRPC Basics: Go](https://grpc.io/docs/tutorials/basic/go.html) for more information.

See the definition of the route guide service in `routeguide/route_guide.proto`.

# Run the sample code
To compile and run the server, assuming you are in the root of the `route_guide`
folder, i.e., `.../examples/route_guide/`, simply:

```sh
$ go run server/server.go --seed 6
```

Likewise, to run the client:

```sh
$ go run client/client.go --addr /ip4/127.0.0.1/tcp/50051/p2p/12D3KooWM4yJB31d4hF2F9Vdwuj9WFo1qonoySyw4bVAQ9a9d21o
```

# Optional command line flags
The server and client both take optional command line flags. For example, the
client and server run without TLS by default. To enable TLS:

```sh
$ go run server/server.go --seed 6 -tls=true
```

and

```sh
$ go run client/client.go --addr /ip4/127.0.0.1/tcp/50051/p2p/12D3KooWM4yJB31d4hF2F9Vdwuj9WFo1qonoySyw4bVAQ9a9d21o -tls=true
```
