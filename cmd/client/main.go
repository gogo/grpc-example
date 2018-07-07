package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/gogo/grpc-example/insecure"
	pbExample "github.com/gogo/grpc-example/proto"
)

var addr = flag.String("addr", "localhost", "The address of the server to connect to")
var port = flag.String("port", "10000", "The port to connect to")

var log grpclog.LoggerV2

func init() {
	log = grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, net.JoinHostPort(*addr, *port),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}
	defer conn.Close()
	c := pbExample.NewUserServiceClient(conn)

	user1 := pbExample.User{ID: 1, Role: pbExample.Role_GUEST}
	_, err = c.AddUser(ctx, &user1)
	if err == nil {
		log.Fatalln("Failed to fail adding user 🤔")
	}
	st := status.Convert(err)
	brErr := st.Details()[0].(*rpc.BadRequest)
	log.Infoln("Failed to create user (as expected):", brErr.String())

	user2 := pbExample.User{ID: 2, Role: pbExample.Role_ADMIN}
	_, err = c.AddUser(ctx, &user2)
	if err != nil {
		log.Fatalln("Failed to add user:", err)
	}

	srv, err := c.ListUsers(ctx, &pbExample.ListUsersRequest{})
	if err != nil {
		log.Fatalln("Failed to list users:", err)
	}
	for {
		rcv, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("Failed to receive:", err)
		}
		log.Infoln("Read user:", rcv)
	}

	log.Infoln("Success!")
}
