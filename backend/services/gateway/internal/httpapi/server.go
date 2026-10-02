// Package httpapi собирает хендлеры фич gateway в единый StrictServerInterface
// и регистрирует их в Gin.
package httpapi

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	"app/backend/services/gateway/internal/gen/api"
	"app/backend/services/gateway/internal/users"
)

// Server реализует api.StrictServerInterface через встраивание хендлеров фич.
type Server struct {
	*users.Handler
}

var _ api.StrictServerInterface = (*Server)(nil)

// Register монтирует публичный API под префиксом /api.
func Register(r gin.IRouter, srv *Server, logger *slog.Logger) {
	strict := api.NewStrictHandlerWithOptions(srv, nil, api.StrictGinServerOptions{
		RequestErrorHandlerFunc: func(c *gin.Context, err error) {
			c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		},
		HandlerErrorFunc: func(c *gin.Context, err error) {
			status := statusFromConnect(err)
			if status >= http.StatusInternalServerError {
				logger.ErrorContext(c, "handler failed", "error", err)
			}
			_ = c.Error(err)
			c.JSON(status, api.Error{Message: http.StatusText(status)})
		},
		ResponseErrorHandlerFunc: func(c *gin.Context, err error) {
			logger.ErrorContext(c, "response failed", "error", err)
			c.JSON(http.StatusInternalServerError, api.Error{Message: http.StatusText(http.StatusInternalServerError)})
		},
	})

	api.RegisterHandlersWithOptions(r, strict, api.GinServerOptions{
		BaseURL: "/api",
		ErrorHandler: func(c *gin.Context, err error, status int) {
			c.JSON(status, api.Error{Message: err.Error()})
		},
	})
}

// statusFromConnect маппит необработанные в хендлерах ошибки сервисов в HTTP-статус.
func statusFromConnect(err error) int {
	switch connect.CodeOf(err) {
	case connect.CodeNotFound:
		return http.StatusNotFound
	case connect.CodeInvalidArgument:
		return http.StatusUnprocessableEntity
	case connect.CodeAlreadyExists:
		return http.StatusConflict
	case connect.CodePermissionDenied:
		return http.StatusForbidden
	case connect.CodeUnauthenticated:
		return http.StatusUnauthorized
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
