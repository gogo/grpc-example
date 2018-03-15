package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/util/protoutil"
	gogoproto "github.com/gogo/protobuf/proto"
	golangproto "github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rakyll/statik/fs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/gogo/grpc-example/insecure"
	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/grpc-example/server"
	// Static files
	_ "github.com/gogo/grpc-example/statik"
)

var (
	gRPCPort    = flag.Int("grpc-port", 10000, "The gRPC server port")
	gatewayPort = flag.Int("gateway-port", 11000, "The gRPC-Gateway server port")
)

var log grpclog.LoggerV2

func init() {
	log = grpclog.NewLoggerV2(os.Stdout, os.Stderr, os.Stderr)
	grpclog.SetLoggerV2(log)
}

// serveOpenAPI serves an OpenAPI UI on /openapi-ui/
// Adapted from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func serveOpenAPI(mux *http.ServeMux) error {
	mime.AddExtensionType(".svg", "image/svg+xml")

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	// Expose files in static on <host>/openapi-ui
	fileServer := http.FileServer(statikFS)
	prefix := "/openapi-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
	return nil
}

// anyResolver is used to implement custom any type resolution
// with the cockroachdb protoutil gogo.JSONPb wrapper.
// This means the JSON marshaller can marshal Any messages
// registered with either gogo/protobuf or golang/protobuf.
type anyResolver struct{}

func (a anyResolver) Resolve(typeURL string) (gogoproto.Message, error) {
	mname := typeURL
	if slash := strings.LastIndex(mname, "/"); slash >= 0 {
		mname = mname[slash+1:]
	}
	// Attempt to use gogo/protobuf resolver
	mt := gogoproto.MessageType(mname)
	if mt == nil {
		// Fallback to golang/protobuf resolver
		mt = golangproto.MessageType(mname)
		if mt == nil {
			// Neither worked, error
			return nil, fmt.Errorf("unknown message type %q", mname)
		}
	}
	return reflect.New(mt.Elem()).Interface().(gogoproto.Message), nil
}

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *gRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	s := grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
		grpc.UnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
	)
	pbExample.RegisterUserServiceServer(s, server.New())

	// Serve gRPC Server
	log.Info("Serving gRPC on https://", addr)
	go func() {
		log.Fatal(s.Serve(lis))
	}()

	// See https://github.com/grpc/grpc/blob/master/doc/naming.md
	// for gRPC naming standard information.
	dialAddr := fmt.Sprintf("ipv4://localhost/%s", addr)
	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	mux := http.NewServeMux()

	jsonpb := &protoutil.JSONPb{
		EmitDefaults: true,
		Indent:       "  ",
		AnyResolver:  anyResolver{},
	}
	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonpb),
	)
	err = pbExample.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	mux.Handle("/", gwmux)
	err = serveOpenAPI(mux)
	if err != nil {
		log.Fatalln("Failed to serve OpenAPI UI")
	}

	gatewayAddr := fmt.Sprintf("localhost:%d", *gatewayPort)
	log.Info("Serving gRPC-Gateway on https://", gatewayAddr)
	log.Info("Serving OpenAPI Documentation on https://", gatewayAddr, "/openapi-ui/")
	gwServer := http.Server{
		Addr: gatewayAddr,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{insecure.Cert},
		},
		Handler: mux,
	}
	log.Fatalln(gwServer.ListenAndServeTLS("", ""))
}
