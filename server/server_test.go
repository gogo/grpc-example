package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/grpc-example/server"
)

//go:generate mockgen -destination mocks_test.go -package server_test github.com/gogo/grpc-example/proto UserService_ListUsersServer,UserService_ListUsersByRoleServer

func TestAddUserListUsers(t *testing.T) {
	b := server.New()
	cd1 := time.Date(2000, 0, 0, 0, 0, 0, 1, time.UTC)
	cd2 := time.Date(2000, 0, 0, 0, 0, 0, 3, time.UTC)
	beforeCD2 := time.Date(2000, 0, 0, 0, 0, 0, 2, time.UTC)
	u1 := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_ADMIN,
		CreateDate: &cd1,
	}
	u2 := &pbExample.User{
		ID:         2,
		Role:       pbExample.Role_GUEST,
		CreateDate: &cd2,
	}
	v, err := b.AddUser(context.Background(), u1)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}
	v, err = b.AddUser(context.Background(), u2)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}

	ctrl := gomock.NewController(t)
	mockServer := NewMockUserService_ListUsersServer(ctrl)
	mockServer.EXPECT().Send(u2).Return(nil)

	err = b.ListUsers(&pbExample.ListUsersRequest{
		CreatedSince: &beforeCD2,
	}, mockServer)
	if err != nil {
		t.Fatal("Failed to list users: ", err)
	}

	ctrl.Finish()
}

func TestAddUserDuplicateFails(t *testing.T) {
	b := server.New()
	cd := time.Date(2000, 0, 0, 0, 0, 0, 1, time.UTC)
	u1 := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_ADMIN,
		CreateDate: &cd,
	}
	v, err := b.AddUser(context.Background(), u1)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}
	_, err = b.AddUser(context.Background(), u1)
	if err == nil {
		t.Fatal("was unexpectedly able to add user twice")
	}
}

func TestAddUserSetsCreateDate(t *testing.T) {
	b := server.New()
	u := &pbExample.User{
		ID:   1,
		Role: pbExample.Role_ADMIN,
	}
	v, err := b.AddUser(context.Background(), u)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}

	ctrl := gomock.NewController(t)
	mockServer := NewMockUserService_ListUsersServer(ctrl)
	mockServer.EXPECT().Send(gomock.Any()).Return(nil).Do(func(sentUser *pbExample.User) {
		if sentUser.GetID() != u.GetID() {
			t.Fatal("Unexpected user ID")
		}
		if sentUser.GetRole() != u.GetRole() {
			t.Fatal("Unexpected user role")
		}
		if sentUser.GetCreateDate() == nil {
			t.Fatal("CreateDate as not set")
		}
		if !sentUser.GetCreateDate().Before(time.Now()) ||
			!sentUser.GetCreateDate().After(time.Now().Add(-time.Second)) {
			t.Fatal("CreateDate was not within the last second: ", sentUser.GetCreateDate())
		}
	})

	err = b.ListUsers(nil, mockServer)
	if err != nil {
		t.Fatal("Failed to list users: ", err)
	}

	ctrl.Finish()
}

func TestListUsersByRole(t *testing.T) {
	b := server.New()
	cd := time.Date(2000, 0, 0, 0, 0, 0, 1, time.UTC)
	admin := &pbExample.User{
		ID:         2,
		Role:       pbExample.Role_ADMIN,
		CreateDate: &cd,
	}
	guest := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_GUEST,
		CreateDate: &cd,
	}
	v, err := b.AddUser(context.Background(), admin)
	if err != nil {
		t.Fatal("Failed to add guest user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}
	v, err = b.AddUser(context.Background(), guest)
	if err != nil {
		t.Fatal("Failed to add admin user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}

	ctrl := gomock.NewController(t)
	mockServer := NewMockUserService_ListUsersByRoleServer(ctrl)
	mockServer.EXPECT().Send(admin).Return(nil)

	err = b.ListUsersByRole(&pbExample.UserRole{Role: pbExample.Role_ADMIN}, mockServer)
	if err != nil {
		t.Fatal("Failed to list users: ", err)
	}

	ctrl.Finish()
}

func TestAddUserNonAdmin(t *testing.T) {
	b := server.New()
	u1 := &pbExample.User{
		ID:   1,
		Role: pbExample.Role_GUEST,
	}
	_, err := b.AddUser(context.Background(), u1)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected error to be a gRPC status, was %#v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Fatalf("Expected error code to be %v, was %v", codes.InvalidArgument, st.Code())
	}

	pb := st.Proto()
	if len(pb.GetDetails()) != 1 {
		t.Fatalf("Expected exactly 1 error detail, was %d", len(pb.GetDetails()))
	}

	br := &rpc.BadRequest{}
	err = proto.Unmarshal(pb.GetDetails()[0].GetValue(), br)
	if err != nil {
		t.Fatalf("Expected error detail to be of type %T, was %s", &rpc.BadRequest{}, pb.GetDetails()[0].GetTypeUrl())
	}

	if len(br.GetFieldViolations()) != 1 {
		t.Fatalf("Expected 1 field violation, was %d", len(br.GetFieldViolations()))
	}

	fv := br.GetFieldViolations()[0]
	if fv.GetField() != "role" {
		t.Fatalf(`Expected field violation to be for "role", was %s`, fv.GetField())
	}
}

func TestListUsersNoUsers(t *testing.T) {
	b := server.New()
	err := b.ListUsers(nil, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected error to be a gRPC status, was %#v", err)
	}

	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("Expected error code to be %v, was %v", codes.FailedPrecondition, st.Code())
	}

	pb := st.Proto()
	if len(pb.GetDetails()) != 1 {
		t.Fatalf("Expected exactly 1 error detail, was %d", len(pb.GetDetails()))
	}

	pf := &rpc.PreconditionFailure{}
	err = proto.Unmarshal(pb.GetDetails()[0].GetValue(), pf)
	if err != nil {
		t.Fatalf("Expected error detail to be of type %T, was %s", &rpc.PreconditionFailure{}, pb.GetDetails()[0].GetTypeUrl())
	}

	if len(pf.GetViolations()) != 1 {
		t.Fatalf("Expected 1 field violation, was %d", len(pf.GetViolations()))
	}

	v := pf.GetViolations()[0]
	if v.GetType() != "USER" {
		t.Fatalf(`Expected field violation to be for "USER", was %s`, v.GetType())
	}
}

func TestUpdateUser(t *testing.T) {
	b := server.New()
	u := &pbExample.User{
		ID:   1,
		Role: pbExample.Role_ADMIN,
	}
	v, err := b.AddUser(context.Background(), u)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	if v == nil {
		t.Fatal("Expected AddUser response not to be nil")
	}

	req := &pbExample.UpdateUserRequest{
		User: &pbExample.User{
			ID:   u.GetID(),
			Role: pbExample.Role_GUEST,
		},
		UpdateMask: &types.FieldMask{
			Paths: []string{"role"},
		},
	}
	newUser, err := b.UpdateUser(context.Background(), req)
	if err != nil {
		t.Fatal("Failed to update user: ", err)
	}

	if newUser.GetRole() != pbExample.Role_GUEST {
		t.Fatalf("Role was not updated to GUEST, was %s", newUser.GetRole().String())
	}

	ctrl := gomock.NewController(t)
	mockServer := NewMockUserService_ListUsersServer(ctrl)
	mockServer.EXPECT().Send(&pbExample.User{
		Role:       pbExample.Role_GUEST,
		ID:         u.GetID(),
		CreateDate: newUser.GetCreateDate(),
	}).Return(nil)

	err = b.ListUsers(nil, mockServer)
	if err != nil {
		t.Fatal("Failed to list users: ", err)
	}

	ctrl.Finish()
}

func TestUpdateMissingUser(t *testing.T) {
	b := server.New()

	req := &pbExample.UpdateUserRequest{
		User: &pbExample.User{
			ID:   1,
			Role: pbExample.Role_GUEST,
		},
		UpdateMask: &types.FieldMask{
			Paths: []string{"role"},
		},
	}
	_, err := b.UpdateUser(context.Background(), req)
	if err == nil {
		t.Fatal("Unexpectedly did not error when updating missing user")
	}

	st := status.Convert(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("Unexpected error code received, got %s expected %s", st.Code().String(), codes.NotFound.String())
	}
}
