// Package httpx — общий bootstrap Gin для всех сервисов.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// ReadyFunc проверяет готовность зависимостей (БД, брокер) для /readyz.
type ReadyFunc func(ctx context.Context) error

// NewRouter создаёт Gin-роутер со стандартными middleware и health-эндпоинтами.
func NewRouter(service string, logger *slog.Logger, ready ReadyFunc) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// *gin.Context передаётся как context.Context (например, в strict-хендлеры oapi-codegen).
	// Без fallback он не видит значения из request-контекста — трейс otelgin бы терялся.
	r.ContextWithFallback = true
	r.Use(gin.Recovery(), otelgin.Middleware(service), requestLogger(logger))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if ready != nil {
			if err := ready(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	return r
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.FullPath() == "/healthz" || c.FullPath() == "/readyz" {
			return
		}
		level := slog.LevelInfo
		if c.Writer.Status() >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}
		logger.Log(c.Request.Context(), level, "http request", attrs...)
	}
}

// Serve запускает HTTP-сервер и корректно останавливает его при отмене ctx.
func Serve(ctx context.Context, addr string, h http.Handler, logger *slog.Logger) error {
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
