// Package rpcx — общие настройки Connect-RPC для серверов и клиентов.
package rpcx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/gin-gonic/gin"
)

// Interceptors возвращает стандартный набор interceptors (трейсинг).
func Interceptors() (connect.Option, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, fmt.Errorf("otelconnect: %w", err)
	}
	return connect.WithInterceptors(otelInterceptor, timeoutInterceptor(5*time.Second)), nil
}

// Mount монтирует Connect-хендлер в Gin-роутер.
func Mount(r gin.IRoutes, path string, h http.Handler) {
	r.Any(path+"*method", gin.WrapH(h))
}

// HTTPClient — клиент для вызовов других сервисов.
func HTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// timeoutInterceptor ставит дедлайн на исходящий вызов, если у контекста его ещё нет.
func timeoutInterceptor(d time.Duration) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := ctx.Deadline(); !ok && req.Spec().IsClient {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, d)
				defer cancel()
			}
			return next(ctx, req)
		}
	}
}
