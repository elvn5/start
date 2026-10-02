package repository

import (
	"errors"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"app/backend/pkg/dbx"
	"app/backend/services/identity/internal/model"
	"app/backend/services/identity/migrations"
)

// Интеграционный тест на реальном Postgres (нужен запущенный Docker).
// Пропускается в режиме `go test -short`.
func TestUsers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := t.Context()

	pg, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("identity"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, pg)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := dbx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := dbx.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}

	repo := NewUsers(pool)

	created, err := repo.Create(ctx, model.NewUser{Email: "ann@example.com", Name: "Ann"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil || got.Email != "ann@example.com" {
		t.Fatalf("get: %+v, %v", got, err)
	}

	if _, err := repo.Create(ctx, model.NewUser{Email: "ann@example.com", Name: "Dup"}); !errors.Is(err, model.ErrEmailTaken) {
		t.Errorf("duplicate email: want ErrEmailTaken, got %v", err)
	}

	if _, err := repo.Get(ctx, 999_999); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("missing user: want ErrNotFound, got %v", err)
	}

	list, err := repo.List(ctx, 10, 0)
	if err != nil || len(list) != 1 {
		t.Errorf("list: %d users, %v", len(list), err)
	}
}
