package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	pbExample "github.com/gogo/grpc-example/proto"
	"github.com/gogo/grpc-example/server"
)

//go:generate mockgen -destination mocks_test.go -package server_test github.com/gogo/grpc-example/proto UserService_ListUsersServer,UserService_ListUsersByRoleServer

func TestAddUserListUsers(t *testing.T) {
	b := server.New()
	cd := time.Date(2000, 0, 0, 0, 0, 0, 1, time.UTC)
	u1 := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_GUEST,
		CreateDate: &cd,
	}
	u2 := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_GUEST,
		CreateDate: &cd,
	}
	_, err := b.AddUser(context.Background(), u1)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}
	_, err = b.AddUser(context.Background(), u2)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
	}

	ctrl := gomock.NewController(t)
	mockServer := NewMockUserService_ListUsersServer(ctrl)
	mockServer.EXPECT().Send(u1).Return(nil)
	mockServer.EXPECT().Send(u2).Return(nil)

	err = b.ListUsers(nil, mockServer)
	if err != nil {
		t.Fatal("Failed to list users: ", err)
	}

	ctrl.Finish()
}

func TestAddUserSetsCreateDate(t *testing.T) {
	b := server.New()
	u := &pbExample.User{
		ID:   1,
		Role: pbExample.Role_GUEST,
	}
	_, err := b.AddUser(context.Background(), u)
	if err != nil {
		t.Fatal("Failed to add user: ", err)
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
	guest := &pbExample.User{
		ID:         1,
		Role:       pbExample.Role_GUEST,
		CreateDate: &cd,
	}
	admin := &pbExample.User{
		ID:         2,
		Role:       pbExample.Role_ADMIN,
		CreateDate: &cd,
	}
	_, err := b.AddUser(context.Background(), guest)
	if err != nil {
		t.Fatal("Failed to add guest user: ", err)
	}
	_, err = b.AddUser(context.Background(), admin)
	if err != nil {
		t.Fatal("Failed to add admin user: ", err)
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
