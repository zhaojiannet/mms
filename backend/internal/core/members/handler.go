package members

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// MemberDTO 对外的会员视图，隐藏 tenant_id（前端无需知道）
type MemberDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Phone     *string   `json:"phone"`
	Gender    string    `json:"gender"`
	Birthday  *string   `json:"birthday"`
	Notes     *string   `json:"notes"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListResponse struct {
	Items []MemberDTO `json:"items"`
	Total int64       `json:"total"`
}

type CreateRequest struct {
	Name     string  `json:"name"`
	Phone    *string `json:"phone"`
	Gender   *string `json:"gender"`
	Birthday *string `json:"birthday"` // YYYY-MM-DD
	Notes    *string `json:"notes"`
}

type UpdateRequest struct {
	Name     *string `json:"name"`
	Phone    *string `json:"phone"`
	Gender   *string `json:"gender"`
	Birthday *string `json:"birthday"`
	Notes    *string `json:"notes"`
	Status   *string `json:"status"`
}

// List GET /api/members?limit=&offset=&status=
func List(c *echo.Context) error {
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	ctx := c.Request().Context()

	limit, offset := paginationFrom(c.Request().URL.Query().Get("limit"), c.Request().URL.Query().Get("offset"))
	var status *string
	if s := c.Request().URL.Query().Get("status"); s != "" {
		status = &s
	}

	rows, err := q.ListMembers(ctx, sqlc.ListMembersParams{Limit: limit, Offset: offset, Status: status})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list members: "+err.Error())
	}
	total, err := q.CountMembers(ctx, status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "count members: "+err.Error())
	}

	items := make([]MemberDTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toDTO(m))
	}
	return c.JSON(http.StatusOK, ListResponse{Items: items, Total: total})
}

// Create POST /api/members
func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	tenant := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	birthday, err := parseDate(req.Birthday)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid birthday: "+err.Error())
	}

	m, err := q.CreateMember(c.Request().Context(), sqlc.CreateMemberParams{
		TenantID: tenant.ID,
		Name:     req.Name,
		Phone:    req.Phone,
		Gender:   req.Gender,
		Birthday: birthday,
		Notes:    req.Notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create member: "+err.Error())
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

// Get GET /api/members/:id
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetMemberByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get member: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Update PUT /api/members/:id
//   - nil 字段保留原值（sqlc.narg + COALESCE）
func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	birthday, err := parseDate(req.Birthday)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid birthday: "+err.Error())
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdateMember(c.Request().Context(), sqlc.UpdateMemberParams{
		ID:       id,
		Name:     req.Name,
		Phone:    req.Phone,
		Gender:   req.Gender,
		Birthday: birthday,
		Notes:    req.Notes,
		Status:   req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update member: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Delete DELETE /api/members/:id
//   - 硬删除；需要软删除时改走 Update(status='inactive')
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	if err := q.DeleteMember(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete member: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.Member) MemberDTO {
	dto := MemberDTO{
		ID:        m.ID,
		Name:      m.Name,
		Phone:     m.Phone,
		Gender:    m.Gender,
		Notes:     m.Notes,
		Status:    m.Status,
		CreatedAt: m.CreatedAt.Time,
		UpdatedAt: m.UpdatedAt.Time,
	}
	if m.Birthday.Valid {
		s := m.Birthday.Time.Format("2006-01-02")
		dto.Birthday = &s
	}
	return dto
}

func parseDate(s *string) (pgtype.Date, error) {
	if s == nil || *s == "" {
		return pgtype.Date{Valid: false}, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func paginationFrom(limitStr, offsetStr string) (int32, int32) {
	limit := int32(50)
	offset := int32(0)
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 200 {
			limit = int32(v)
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = int32(v)
		}
	}
	return limit, offset
}
