package server

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbExample "github.com/gogo/grpc-example/proto"
)

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

	if len(b.users) == 0 && user.GetRole() != pbExample.Role_ADMIN {
		st := status.New(codes.InvalidArgument, "first user created must be an admin")
		// Note that st.WithDetails requires a proto.Message that has been registered
		// with golang/protobuf to work. This in turn requires us to instrument our
		// jsonpb Marshaller such that it can resolve both gogo/protobuf and golang/protobuf Any messages.
		detSt, err := st.WithDetails(&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "role",
					Description: "The first user created must have the role of an ADMIN",
				},
			},
		})
		if err == nil {
			return nil, detSt.Err()
		}

		return nil, st.Err()
	}

	if user.GetCreateDate() == nil {
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
