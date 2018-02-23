# gRPC-Example

This repo is an example of using [Go gRPC](https://github.com/grpc/grpc-go)
and tools from the greater gRPC ecosystem together with  the
[GoGo Protobuf Project](https://github.com/gogo/protobuf).

## Running it

```bash
$ go run main.go
INFO: Serving gRPC on https://localhost:10000
INFO: dialing to target with scheme: "ipv4"
INFO: ccResolverWrapper: sending new addresses to cc: [{localhost:10000 0  <nil>}]
INFO: ClientConn switching balancer to "pick_first"
INFO: pickfirstBalancer: HandleSubConnStateChange: 0xc420097c00, CONNECTING
INFO: pickfirstBalancer: HandleSubConnStateChange: 0xc420097c00, READY
INFO: Serving gRPC-Gateway on https://localhost:11000
INFO: Serving OpenAPI Documentation on https://localhost:11000/openapi-ui/
```

After starting the server, you can access the OpenAPI UI on
[https://localhost:11000/openapi-ui/](https://localhost:11000/openapi-ui/)

## Development

To regenerate the proto files, ensure you have installed the generate dependencies:

```bash
$ go install ./vendor/...
```

It also requires you to have the Google Protobuf compiler `protoc` installed.
Please follow instructions for your platform on the
[official protoc repo](https://github.com/google/protobuf#protocol-compiler-installation).

Regenerate the files by running `go generate`:

```bash
$ go generate ./server/
```
