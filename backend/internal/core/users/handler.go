// Package users 管理后台用户（员工账号）的 CRUD
// 老的 /users/password（自助改密码）留在 auth 包下
package users

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	Phone       *string    `json:"phone"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRequest struct {
	Email    string  `json:"email"`
	Phone    *string `json:"phone"`
	Name     string  `json:"name"`
	Role     string  `json:"role"` // admin / manager / staff
	Password string  `json:"password"`
}

type UpdateRequest struct {
	Name   *string `json:"name"`
	Phone  *string `json:"phone"`
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func List(c *echo.Context) error {
	q := sqlc.New(mw.TxFrom(c))
	var status *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}
	rows, err := q.ListUsersByTenant(c.Request().Context(), sqlc.ListUsersByTenantParams{
		Status: status, Limit: 200, Offset: 0,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list: "+err.Error())
	}
	items := make([]DTO, 0, len(rows))
	for _, u := range rows {
		items = append(items, toDTO(u))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	q := sqlc.New(mw.TxFrom(c))
	u, err := q.GetUserByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(u))
}

func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Email == "" || req.Name == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email / name / password required")
	}
	if len(req.Password) < 6 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 6 chars")
	}
	role := req.Role
	if role == "" {
		role = "staff"
	}
	if role != "admin" && role != "manager" && role != "staff" {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be admin/manager/staff")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "hash: "+err.Error())
	}

	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	u, err := q.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
		TenantID:     t.ID,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hash,
		Name:         req.Name,
		Role:         &role,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}
	return c.JSON(http.StatusCreated, toDTO(u))
}

func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Role != nil && *req.Role != "admin" && *req.Role != "manager" && *req.Role != "staff" {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be admin/manager/staff")
	}
	q := sqlc.New(mw.TxFrom(c))
	u, err := q.UpdateUser(c.Request().Context(), sqlc.UpdateUserParams{
		ID: id, Name: req.Name, Phone: req.Phone, Role: req.Role, Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(u))
}

func ResetPassword(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if len(req.NewPassword) < 6 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 6 chars")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "hash: "+err.Error())
	}
	q := sqlc.New(mw.TxFrom(c))
	if err := q.UpdateUserPassword(c.Request().Context(), sqlc.UpdateUserPasswordParams{
		ID: id, PasswordHash: hash,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update password: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	// 不允许删自己
	claims := mw.ClaimsFrom(c)
	if claims.UserID == id {
		return echo.NewHTTPError(http.StatusBadRequest, "不能删除自己")
	}
	q := sqlc.New(mw.TxFrom(c))
	if err := q.DeleteUser(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(u sqlc.User) DTO {
	dto := DTO{
		ID:        u.ID,
		Email:     u.Email,
		Phone:     u.Phone,
		Name:      u.Name,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Time,
	}
	if u.LastLoginAt.Valid {
		v := u.LastLoginAt.Time
		dto.LastLoginAt = &v
	}
	return dto
}
