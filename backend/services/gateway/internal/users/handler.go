// Package users — фича «пользователи» в gateway: REST (OpenAPI) → identity (Connect).
package users

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	openapi_types "github.com/oapi-codegen/runtime/types"

	identityv1 "app/backend/gen/proto/identity/v1"
	"app/backend/gen/proto/identity/v1/identityv1connect"
	"app/backend/services/gateway/internal/gen/api"
)

type Handler struct {
	identity identityv1connect.IdentityServiceClient
}

func NewHandler(identity identityv1connect.IdentityServiceClient) *Handler {
	return &Handler{identity: identity}
}

func (h *Handler) ListUsers(ctx context.Context, req api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	msg := &identityv1.ListUsersRequest{}
	if req.Params.Limit != nil {
		msg.Limit = int32(*req.Params.Limit) //nolint:gosec // ограничено спекой (≤100)
	}
	if req.Params.Offset != nil {
		msg.Offset = int32(*req.Params.Offset) //nolint:gosec
	}

	res, err := h.identity.ListUsers(ctx, connect.NewRequest(msg))
	if err != nil {
		return nil, fmt.Errorf("identity.ListUsers: %w", err)
	}
	out := make(api.ListUsers200JSONResponse, 0, len(res.Msg.GetUsers()))
	for _, u := range res.Msg.GetUsers() {
		out = append(out, toAPI(u))
	}
	return out, nil
}

func (h *Handler) CreateUser(ctx context.Context, req api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	res, err := h.identity.CreateUser(ctx, connect.NewRequest(&identityv1.CreateUserRequest{
		Email: string(req.Body.Email),
		Name:  req.Body.Name,
	}))
	switch connect.CodeOf(err) {
	case connect.CodeAlreadyExists:
		return api.CreateUser409JSONResponse{ErrorJSONResponse: api.ErrorJSONResponse{Message: "email already taken"}}, nil
	case connect.CodeInvalidArgument:
		return api.CreateUser422JSONResponse{Message: connectMessage(err)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("identity.CreateUser: %w", err)
	}
	return api.CreateUser201JSONResponse(toAPI(res.Msg.GetUser())), nil
}

func (h *Handler) GetUser(ctx context.Context, req api.GetUserRequestObject) (api.GetUserResponseObject, error) {
	res, err := h.identity.GetUser(ctx, connect.NewRequest(&identityv1.GetUserRequest{Id: req.Id}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return api.GetUser404JSONResponse{ErrorJSONResponse: api.ErrorJSONResponse{Message: "user not found"}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("identity.GetUser: %w", err)
	}
	return api.GetUser200JSONResponse(toAPI(res.Msg.GetUser())), nil
}

func toAPI(u *identityv1.User) api.User {
	return api.User{
		Id:        u.GetId(),
		Email:     openapi_types.Email(u.GetEmail()),
		Name:      u.GetName(),
		CreatedAt: u.GetCreatedAt().AsTime(),
	}
}

func connectMessage(err error) string {
	if ce, ok := errors.AsType[*connect.Error](err); ok {
		return ce.Message()
	}
	return "invalid input"
}
