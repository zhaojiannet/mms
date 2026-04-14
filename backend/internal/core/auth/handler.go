package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	pauth "github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        UserDTO   `json:"user"`
}

type UserDTO struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

// LoginHandler POST /api/login
//   - 依赖 TenantResolver + TenantTx 中间件（已 SET LOCAL app.current_tenant）
//   - RLS 保证 GetUserByEmail 只能看到本租户用户
func LoginHandler(c *echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password required")
	}

	tenant := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	queries := sqlc.New(tx)

	ctx := c.Request().Context()
	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "lookup user: "+err.Error())
	}
	if user.Status != "active" {
		return echo.NewHTTPError(http.StatusForbidden, "user is "+user.Status)
	}

	ok, err := pauth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "verify password: "+err.Error())
	}
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	if err := queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		// 非致命，不中断登录
		slog.Warn("update last_login_at failed", "user_id", user.ID, "err", err)
	}

	token, expiresAt, err := pauth.SignAccessToken(user.ID, tenant.ID, user.Email, user.Role)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "sign token: "+err.Error())
	}

	return c.JSON(http.StatusOK, LoginResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt,
		User: UserDTO{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Role:  user.Role,
		},
	})
}
