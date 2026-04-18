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
	Role     string  `json:"role"` // super_admin / admin / staff
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
		return mw.InternalError(c, "list: ", err)
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
		return mw.InternalError(c, "get: ", err)
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
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 chars")
	}
	role := req.Role
	if role == "" {
		role = mw.RoleStaff
	}
	if role != mw.RoleSuperAdmin && role != mw.RoleAdmin && role != mw.RoleStaff {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be super_admin/admin/staff")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return mw.InternalError(c, "hash: ", err)
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
		return mw.InternalError(c, "create: ", err)
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
	if req.Role != nil && *req.Role != mw.RoleSuperAdmin && *req.Role != mw.RoleAdmin && *req.Role != mw.RoleStaff {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be super_admin/admin/staff")
	}

	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	// 最后一个超级管理员保护：原是 active super_admin 的用户，修改会导致其不再是 active super_admin 时，
	// 必须确保系统内至少还有另一个 active super_admin
	cur, err := q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return mw.InternalError(c, "get user: ", err)
	}
	willLeaveSuper := cur.Role == mw.RoleSuperAdmin && cur.Status == "active" &&
		((req.Role != nil && *req.Role != mw.RoleSuperAdmin) ||
			(req.Status != nil && *req.Status != "active"))
	if willLeaveSuper {
		// SELECT ... FOR UPDATE 锁定所有 active super_admin 行，防 TOCTOU 竞态
		// 其他并发的"降级/删除"事务会在这里阻塞，直到本事务 commit/rollback
		ids, err := q.LockActiveSuperAdmins(ctx)
		if err != nil {
			return mw.InternalError(c, "lock super admins: ", err)
		}
		if len(ids) <= 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "本店必须至少保留 1 位超级管理员，请先将其他用户升为超级管理员")
		}
	}

	u, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID: id, Name: req.Name, Phone: req.Phone, Role: req.Role, Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return mw.InternalError(c, "update: ", err)
	}
	// role 或 status 变动 → 递增 token_version，持有旧 token 的会话立即失效
	roleChanged := req.Role != nil && *req.Role != cur.Role
	statusChanged := req.Status != nil && *req.Status != cur.Status
	if roleChanged || statusChanged {
		if err := q.IncrementUserTokenVersion(ctx, id); err != nil {
			return mw.InternalError(c, "bump token version: ", err)
		}
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
	if len(req.NewPassword) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 chars")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return mw.InternalError(c, "hash: ", err)
	}
	q := sqlc.New(mw.TxFrom(c))
	if err := q.UpdateUserPassword(c.Request().Context(), sqlc.UpdateUserPasswordParams{
		ID: id, PasswordHash: hash,
	}); err != nil {
		return mw.InternalError(c, "update password: ", err)
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

	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	// 最后一个超级管理员保护：目标若是 active super_admin，系统必须还有其他 active super_admin
	cur, err := q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return mw.InternalError(c, "get user: ", err)
	}
	if cur.Role == mw.RoleSuperAdmin && cur.Status == "active" {
		ids, err := q.LockActiveSuperAdmins(ctx)
		if err != nil {
			return mw.InternalError(c, "lock super admins: ", err)
		}
		if len(ids) <= 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "本店必须至少保留 1 位超级管理员，请先将其他用户升为超级管理员")
		}
	}

	if err := q.DeleteUser(ctx, id); err != nil {
		return mw.InternalError(c, "delete: ", err)
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
