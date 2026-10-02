package httpapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "app/backend/gen/proto/identity/v1"
	"app/backend/gen/proto/identity/v1/identityv1connect"
	"app/backend/pkg/httpx"
	"app/backend/services/gateway/internal/httpapi"
	"app/backend/services/gateway/internal/users"
)

// fakeIdentity — поддельный identity-сервис, поднятый как настоящий Connect-сервер.
type fakeIdentity struct {
	identityv1connect.UnimplementedIdentityServiceHandler
}

func (fakeIdentity) GetUser(_ context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	if req.Msg.GetId() != 1 {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{User: &identityv1.User{
		Id: 1, Email: "ann@example.com", Name: "Ann", CreatedAt: timestamppb.New(time.Unix(0, 0)),
	}}), nil
}

func (fakeIdentity) CreateUser(_ context.Context, req *connect.Request[identityv1.CreateUserRequest]) (*connect.Response[identityv1.CreateUserResponse], error) {
	return nil, connect.NewError(connect.CodeAlreadyExists, nil)
}

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	path, h := identityv1connect.NewIdentityServiceHandler(fakeIdentity{})
	mux := http.NewServeMux()
	mux.Handle(path, h)
	identitySrv := httptest.NewServer(mux)
	t.Cleanup(identitySrv.Close)

	client := identityv1connect.NewIdentityServiceClient(identitySrv.Client(), identitySrv.URL)
	logger := slog.New(slog.DiscardHandler)
	r := httpx.NewRouter("gateway-test", logger, nil)
	httpapi.Register(r, &httpapi.Server{Handler: users.NewHandler(client)}, logger)
	return r
}

func TestGateway(t *testing.T) {
	gw := newGateway(t)

	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantInBody               string
	}{
		{"get existing user", http.MethodGet, "/api/users/1", "", http.StatusOK, `"email":"ann@example.com"`},
		{"get missing user", http.MethodGet, "/api/users/2", "", http.StatusNotFound, `"message":"user not found"`},
		{"invalid id", http.MethodGet, "/api/users/abc", "", http.StatusBadRequest, `"message"`},
		{"duplicate email", http.MethodPost, "/api/users", `{"email":"ann@example.com","name":"Ann"}`, http.StatusConflict, `"message":"email already taken"`},
		{"unimplemented rpc → 500", http.MethodGet, "/api/users", "", http.StatusInternalServerError, `"message"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			gw.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d (%s)", tt.wantStatus, rec.Code, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.wantInBody) {
				t.Errorf("body %q does not contain %q", rec.Body, tt.wantInBody)
			}
			if !json.Valid(rec.Body.Bytes()) {
				t.Errorf("body is not valid JSON: %s", rec.Body)
			}
		})
	}
}
