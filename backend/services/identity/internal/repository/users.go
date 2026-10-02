// Package repository — доступ к данным identity через sqlc.
// Типы sqlc (gen/db) не выходят за пределы этого пакета.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"app/backend/services/identity/internal/gen/db"
	"app/backend/services/identity/internal/model"
)

const uniqueViolation = "23505"

type Users struct{ q *db.Queries }

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{q: db.New(pool)}
}

func (r *Users) Get(ctx context.Context, id int64) (model.User, error) {
	u, err := r.q.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return toModel(u), nil
}

func (r *Users) List(ctx context.Context, limit, offset int32) ([]model.User, error) {
	rows, err := r.q.ListUsers(ctx, db.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]model.User, 0, len(rows))
	for _, u := range rows {
		users = append(users, toModel(u))
	}
	return users, nil
}

func (r *Users) Create(ctx context.Context, in model.NewUser) (model.User, error) {
	u, err := r.q.CreateUser(ctx, db.CreateUserParams{Email: in.Email, Name: in.Name})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolation {
		return model.User{}, model.ErrEmailTaken
	}
	if err != nil {
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return toModel(u), nil
}

func toModel(u db.User) model.User {
	return model.User{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt}
}
