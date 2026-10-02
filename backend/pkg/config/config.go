// Package config загружает конфигурацию сервиса из переменных окружения.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Load заполняет T из env-переменных с префиксом (например, "IDENTITY_").
func Load[T any](prefix string) (T, error) {
	cfg, err := env.ParseAsWithOptions[T](env.Options{Prefix: prefix})
	if err != nil {
		return cfg, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
