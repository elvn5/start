// Package migrations встраивает SQL-миграции сервиса в бинарь.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
