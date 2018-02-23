package main

import (
	"context"
	"flag"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/util/protoutil"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/grpc-example/server"
)

var (
	certFile    = flag.String("cert_file", "./insecure/cert.pem", "The TLS cert file")
	keyFile     = flag.String("key_file", "./insecure/key.pem", "The TLS key file")
	gRPCPort    = flag.Int("grpc-port", 10000, "The gRPC server port")
	gatewayPort = flag.Int("gateway-port", 11000, "The gRPC-Gateway server port")
)

var log grpclog.LoggerV2

func init() {
	log = grpclog.NewLoggerV2(os.Stdout, os.Stderr, os.Stderr)
	grpclog.SetLoggerV2(log)
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

// serveOpenAPI serves an OpenAPI UI on /openapi-ui/
// Adapted  from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func serveOpenAPI(mux *http.ServeMux) {
	mime.AddExtensionType(".svg", "image/svg+xml")

	// Expose files in third_party/OpenAPI/ on <host>/openapi-ui
	fileServer := http.FileServer(http.Dir("./third_party/OpenAPI"))
	prefix := "/openapi-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}

func main() {
	flag.Parse()
	addr := fmt.Sprintf("localhost:%d", *gRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	serverCreds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
	if err != nil {
		log.Fatalln("Failed to generate server credentials:", err)
	}
	s := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.UnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
	)
	pbExample.RegisterUserServiceServer(s, server.New())

	// Serve gRPC Server
	log.Info("Serving gRPC on https://", addr)
	go func() {
		log.Fatal(s.Serve(lis))
	}()

	clientCreds, err := credentials.NewClientTLSFromFile(*certFile, "")
	if err != nil {
		log.Fatalln("Failed to generate client credentials:", err)
	}

	// See https://github.com/grpc/grpc/blob/master/doc/naming.md
	// for gRPC naming standard information.
	dialAddr := fmt.Sprintf("ipv4://localhost/%[1]s", addr)
	conn, err := grpc.DialContext(
		context.Background(),
		dialAddr,
		grpc.WithTransportCredentials(clientCreds),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	mux := http.NewServeMux()

	jsonpb := &protoutil.JSONPb{
		EmitDefaults: true,
		Indent:       "  ",
	}
	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonpb),
	)
	err = pbExample.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	mux.Handle("/", gwmux)
	serveOpenAPI(mux)

	gatewayAddr := fmt.Sprintf("localhost:%d", *gatewayPort)
	log.Info("Serving gRPC-Gateway on https://", gatewayAddr)
	log.Info("Serving OpenAPI Documentation on https://", gatewayAddr, "/openapi-ui/")
	http.ListenAndServeTLS(gatewayAddr, *certFile, *keyFile, mux)
}
