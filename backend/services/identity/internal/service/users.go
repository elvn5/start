// Package service — бизнес-логика identity. Ничего не знает о Gin, Connect и БД.
package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"app/backend/services/identity/internal/model"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	maxNameLen   = 100
)

// usersRepo — то, что сервису нужно от хранилища (интерфейс объявляет потребитель).
type usersRepo interface {
	Get(ctx context.Context, id int64) (model.User, error)
	List(ctx context.Context, limit, offset int32) ([]model.User, error)
	Create(ctx context.Context, in model.NewUser) (model.User, error)
}

type Users struct{ repo usersRepo }

func NewUsers(repo usersRepo) *Users {
	return &Users{repo: repo}
}

func (s *Users) Get(ctx context.Context, id int64) (model.User, error) {
	if id <= 0 {
		return model.User{}, fmt.Errorf("%w: id must be positive", model.ErrInvalid)
	}
	return s.repo.Get(ctx, id)
}

func (s *Users) List(ctx context.Context, limit, offset int32) ([]model.User, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)
	offset = max(offset, 0)
	return s.repo.List(ctx, limit, offset)
}

func (s *Users) Create(ctx context.Context, in model.NewUser) (model.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)

	if _, err := mail.ParseAddress(in.Email); err != nil {
		return model.User{}, fmt.Errorf("%w: invalid email", model.ErrInvalid)
	}
	if in.Name == "" || len([]rune(in.Name)) > maxNameLen {
		return model.User{}, fmt.Errorf("%w: name must be 1..%d characters", model.ErrInvalid, maxNameLen)
	}
	return s.repo.Create(ctx, in)
}
