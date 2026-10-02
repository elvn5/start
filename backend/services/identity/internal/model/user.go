// Package model — доменные модели и ошибки сервиса identity.
package model

import (
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already taken")
	ErrInvalid    = errors.New("invalid input")
)

type User struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
}

type NewUser struct {
	Email string
	Name  string
}
