package server

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

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
		st := status.New(codes.InvalidArgument, "First user created must be an admin")
		detSt, err := st.WithDetails(&rpc.BadRequest{
			FieldViolations: []*rpc.BadRequest_FieldViolation{
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

	// Check user ID doesn't already exist
	for _, u := range b.users {
		if u.GetID() == user.GetID() {
			return nil, status.Error(codes.FailedPrecondition, "user already exists")
		}
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

	if len(b.users) == 0 {
		st := status.New(codes.FailedPrecondition, "No users have been created")
		detSt, err := st.WithDetails(&rpc.PreconditionFailure{
			Violations: []*rpc.PreconditionFailure_Violation{
				{
					Type:        "USER",
					Subject:     "no users created",
					Description: "No users have been created",
				},
			},
		})
		if err == nil {
			return detSt.Err()
		}
		return st.Err()
	}

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
