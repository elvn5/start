package service

import (
	"context"
	"errors"
	"testing"

	"app/backend/services/identity/internal/model"
)

type fakeRepo struct {
	created   model.NewUser
	listLimit int32
}

func (f *fakeRepo) Get(_ context.Context, id int64) (model.User, error) {
	if id == 42 {
		return model.User{ID: 42}, nil
	}
	return model.User{}, model.ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, limit, _ int32) ([]model.User, error) {
	f.listLimit = limit
	return nil, nil
}

func (f *fakeRepo) Create(_ context.Context, in model.NewUser) (model.User, error) {
	f.created = in
	return model.User{ID: 1, Email: in.Email, Name: in.Name}, nil
}

func TestCreate_NormalizesInput(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewUsers(repo)

	_, err := svc.Create(t.Context(), model.NewUser{Email: "  Ann@Example.COM ", Name: " Ann "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created.Email != "ann@example.com" || repo.created.Name != "Ann" {
		t.Errorf("input not normalized: %+v", repo.created)
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := map[string]model.NewUser{
		"bad email":  {Email: "not-an-email", Name: "Ann"},
		"empty name": {Email: "ann@example.com", Name: "   "},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewUsers(&fakeRepo{}).Create(t.Context(), in)
			if !errors.Is(err, model.ErrInvalid) {
				t.Errorf("want ErrInvalid, got %v", err)
			}
		})
	}
}

func TestList_ClampsLimit(t *testing.T) {
	tests := []struct{ in, want int32 }{{0, defaultLimit}, {-5, defaultLimit}, {50, 50}, {1000, maxLimit}}
	for _, tt := range tests {
		repo := &fakeRepo{}
		if _, err := NewUsers(repo).List(t.Context(), tt.in, 0); err != nil {
			t.Fatal(err)
		}
		if repo.listLimit != tt.want {
			t.Errorf("limit %d: want %d, got %d", tt.in, tt.want, repo.listLimit)
		}
	}
}

func TestGet_RejectsNonPositiveID(t *testing.T) {
	_, err := NewUsers(&fakeRepo{}).Get(t.Context(), 0)
	if !errors.Is(err, model.ErrInvalid) {
		t.Errorf("want ErrInvalid, got %v", err)
	}
}
