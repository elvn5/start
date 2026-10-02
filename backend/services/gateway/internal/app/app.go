// Package app собирает зависимости gateway (ручной DI) и запускает сервис.
package app

import (
	"context"
	"fmt"

	"app/backend/gen/proto/identity/v1/identityv1connect"
	"app/backend/pkg/config"
	"app/backend/pkg/httpx"
	"app/backend/pkg/observability"
	"app/backend/pkg/rpcx"
	"app/backend/services/gateway/internal/httpapi"
	"app/backend/services/gateway/internal/users"
)

const name = "gateway"

type Config struct {
	Addr        string `env:"ADDR" envDefault:":8080"`
	IdentityURL string `env:"IDENTITY_URL" envDefault:"http://localhost:8081"`
}

func Run(ctx context.Context) error {
	cfg, err := config.Load[Config]("GATEWAY_")
	if err != nil {
		return err
	}

	logger, shutdown, err := observability.Setup(ctx, name)
	if err != nil {
		return err
	}
	defer shutdown(context.WithoutCancel(ctx)) //nolint:errcheck

	interceptors, err := rpcx.Interceptors()
	if err != nil {
		return err
	}
	identity := identityv1connect.NewIdentityServiceClient(rpcx.HTTPClient(), cfg.IdentityURL, interceptors)

	srv := &httpapi.Server{
		Handler: users.NewHandler(identity),
	}

	r := httpx.NewRouter(name, logger, nil)
	httpapi.Register(r, srv, logger)

	if err := httpx.Serve(ctx, cfg.Addr, r, logger); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
