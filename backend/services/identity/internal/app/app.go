// Package app собирает зависимости identity (ручной DI) и запускает сервис.
package app

import (
	"context"
	"fmt"

	"app/backend/gen/proto/identity/v1/identityv1connect"
	"app/backend/pkg/config"
	"app/backend/pkg/dbx"
	"app/backend/pkg/httpx"
	"app/backend/pkg/observability"
	"app/backend/pkg/rpcx"
	"app/backend/services/identity/internal/repository"
	"app/backend/services/identity/internal/rpc"
	"app/backend/services/identity/internal/service"
	"app/backend/services/identity/migrations"
)

const name = "identity"

type Config struct {
	Addr        string `env:"ADDR" envDefault:":8081"`
	DatabaseURL string `env:"DATABASE_URL,required"`
}

func Run(ctx context.Context) error {
	cfg, err := config.Load[Config]("IDENTITY_")
	if err != nil {
		return err
	}

	logger, shutdown, err := observability.Setup(ctx, name)
	if err != nil {
		return err
	}
	defer shutdown(context.WithoutCancel(ctx)) //nolint:errcheck

	pool, err := dbx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := dbx.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}

	users := service.NewUsers(repository.NewUsers(pool))

	interceptors, err := rpcx.Interceptors()
	if err != nil {
		return err
	}
	path, handler := identityv1connect.NewIdentityServiceHandler(rpc.NewServer(users), interceptors)

	r := httpx.NewRouter(name, logger, func(ctx context.Context) error { return pool.Ping(ctx) })
	rpcx.Mount(r, path, handler)

	if err := httpx.Serve(ctx, cfg.Addr, r, logger); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
