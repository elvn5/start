// Package rpc — транспортный слой identity: Connect-хендлеры.
// Protobuf-типы не выходят за пределы этого пакета.
package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "app/backend/gen/proto/identity/v1"
	"app/backend/gen/proto/identity/v1/identityv1connect"
	"app/backend/services/identity/internal/model"
	"app/backend/services/identity/internal/service"
)

type Server struct{ users *service.Users }

var _ identityv1connect.IdentityServiceHandler = (*Server)(nil)

func NewServer(users *service.Users) *Server {
	return &Server{users: users}
}

func (s *Server) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	u, err := s.users.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{User: toProto(u)}), nil
}

func (s *Server) ListUsers(ctx context.Context, req *connect.Request[identityv1.ListUsersRequest]) (*connect.Response[identityv1.ListUsersResponse], error) {
	users, err := s.users.List(ctx, req.Msg.GetLimit(), req.Msg.GetOffset())
	if err != nil {
		return nil, toConnectError(err)
	}
	out := make([]*identityv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, toProto(u))
	}
	return connect.NewResponse(&identityv1.ListUsersResponse{Users: out}), nil
}

func (s *Server) CreateUser(ctx context.Context, req *connect.Request[identityv1.CreateUserRequest]) (*connect.Response[identityv1.CreateUserResponse], error) {
	u, err := s.users.Create(ctx, model.NewUser{Email: req.Msg.GetEmail(), Name: req.Msg.GetName()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&identityv1.CreateUserResponse{User: toProto(u)}), nil
}

func toConnectError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, model.ErrEmailTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, model.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return err // Connect вернёт CodeUnknown; детали не уходят клиенту в сообщении кода
	}
}

func toProto(u model.User) *identityv1.User {
	return &identityv1.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}
