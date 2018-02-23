package server

import (
	"context"
	"sync"
	"time"

	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/protobuf/types"
)

// Generate gogo, grpc-gateway, swagger, validators
//go:generate protoc -I ../proto -I ../vendor/github.com/grpc-ecosystem/grpc-gateway/ -I ../vendor/github.com/gogo/googleapis/ -I ../vendor/ --gogo_out=plugins=grpc,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:$GOPATH/src/ --grpc-gateway_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:$GOPATH/src/ --swagger_out=../third_party/OpenAPI/ --govalidators_out=gogoimport=true,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api:$GOPATH/src ../proto/example.proto

// Workaround for https://github.com/grpc-ecosystem/grpc-gateway/issues/229.
//go:generate sed -i "s/empty.Empty/types.Empty/g" ../proto/example.pb.gw.go

type Backend struct {
	mu    *sync.RWMutex
	users []*pbExample.User
}

var _ pbExample.UserServiceServer = (*Backend)(nil)

func New() *Backend {
	return &Backend{
		mu: &sync.RWMutex{},
	}
}

func (b *Backend) AddUser(ctx context.Context, user *pbExample.User) (*types.Empty, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if user.GetCreateDate == nil {
		now := time.Now()
		user.CreateDate = &now
	}
	b.users = append(b.users, user)
	return nil, nil
}

func (b *Backend) ListUsers(_ *types.Empty, srv pbExample.UserService_ListUsersServer) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, user := range b.users {
		err := srv.Send(user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Backend) ListUsersByRole(req *pbExample.UserRole, srv pbExample.UserService_ListUsersByRoleServer) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, user := range b.users {
		if user.GetRole() == req.GetRole() {
			err := srv.Send(user)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
