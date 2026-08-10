package members

import (
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/core/quota"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/timex"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// MemberDTO 对外的会员视图，隐藏 tenant_id（前端无需知道）
// 列表场景会附带聚合字段（TotalBalance/TotalPending/CardCount），单查/创建/更新时保持 0
type MemberDTO struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Phone           *string   `json:"phone"`
	Gender          string    `json:"gender"`
	Birthday        *string   `json:"birthday"`
	Notes           *string   `json:"notes"`
	Status          string    `json:"status"`
	TotalBalance    string    `json:"total_balance"`
	TotalPending    string    `json:"total_pending"`
	CardCount       int64     `json:"card_count"`
	ActiveCardCount int64     `json:"active_card_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ListResponse struct {
	Items []MemberDTO `json:"items"`
	Total int64       `json:"total"`
	// 租户级聚合标志：挂账/备注列的显隐依据（分页切片推导会随翻页跳变）
	HasPending bool `json:"has_pending"`
	HasNotes   bool `json:"has_notes"`
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
	var search *string
	if s := c.Request().URL.Query().Get("search"); s != "" {
		// 上限 100：防 ILIKE '%超长字符串%' 全表 CPU 飙升（自伤型 DoS）
		if utf8.RuneCountInString(s) > 100 {
			return echo.NewHTTPError(http.StatusBadRequest, "搜索关键词过长（最多 100 字）")
		}
		search = &s
	}

	// staff 角色的查询限制：防止批量盗取会员信息
	//   - 必须提供 search 参数且长度 >= 3
	//   - limit 硬上限 20
	claims := mw.ClaimsFrom(c)
	if claims.Role == mw.RoleStaff {
		if search == nil || utf8.RuneCountInString(*search) < 3 {
			return echo.NewHTTPError(http.StatusBadRequest, "请输入至少 3 个字符进行搜索")
		}
		if limit > 20 {
			limit = 20
		}
	}

	rows, err := q.ListMembers(ctx, sqlc.ListMembersParams{
		Limit: limit, Offset: offset, Status: status, Search: search,
	})
	if err != nil {
		return mw.InternalError(c, "list members: ", err)
	}
	total, err := q.CountMembers(ctx, sqlc.CountMembersParams{Status: status, Search: search})
	if err != nil {
		return mw.InternalError(c, "count members: ", err)
	}

	items := make([]MemberDTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toListDTO(m))
	}
	hasPending, err := q.TenantHasPendingCredits(ctx)
	if err != nil {
		return mw.InternalError(c, "members has_pending: ", err)
	}
	hasNotes, err := q.TenantHasMemberNotes(ctx)
	if err != nil {
		return mw.InternalError(c, "members has_notes: ", err)
	}
	return c.JSON(http.StatusOK, ListResponse{Items: items, Total: total, HasPending: hasPending, HasNotes: hasNotes})
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
	if req.Phone == nil || *req.Phone == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "phone is required")
	}
	if err := quota.Enforce(c, quota.KindMembers); err != nil {
		return err
	}

	tenant := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	birthday, err := timex.ParseDate(req.Birthday)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid birthday: "+err.Error())
	}

	if err := checkPhoneUnique(c, q, *req.Phone, nil); err != nil {
		return err
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
		return mw.InternalError(c, "create member: ", err)
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
		return mw.InternalError(c, "get member: ", err)
	}
	dto := toDTO(m)
	// POS 经 ?member= 入口直接用详情接口的余额/挂账，必须是真实聚合而非占位 0
	stats, err := q.GetMemberStats(c.Request().Context(), id)
	if err != nil {
		return mw.InternalError(c, "get member stats: ", err)
	}
	dto.TotalBalance = stats.TotalBalance.String()
	dto.TotalPending = stats.TotalPending.String()
	dto.CardCount = stats.CardCount
	dto.ActiveCardCount = stats.ActiveCardCount
	return c.JSON(http.StatusOK, dto)
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

	birthday, err := timex.ParseDate(req.Birthday)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid birthday: "+err.Error())
	}
	// 恢复 active 按新增计限额（堵"归档→新建→恢复"绕过）；排除自身让已 active 的原地编辑不受限
	if req.Status != nil && *req.Status == "active" {
		if err := quota.EnforceOnActivate(c, quota.KindMembers, id); err != nil {
			return err
		}
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	if req.Phone != nil && *req.Phone != "" {
		if err := checkPhoneUnique(c, q, *req.Phone, &id); err != nil {
			return err
		}
	}
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
		return mw.InternalError(c, "update member: ", err)
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
	ctx := c.Request().Context()
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	// 有资金痕迹的会员只能停用不能删：cards.member_id 是 ON DELETE CASCADE，
	// 物理删除会连带删卡，进而触碰资金流水（00031 起 RESTRICT 会直接报 FK 错）
	cards, err := q.ListCardsByMember(ctx, id)
	if err != nil {
		return mw.InternalError(c, "delete member.check_cards: ", err)
	}
	if len(cards) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "该会员名下有会员卡，不能删除，建议改为停用")
	}
	includeVoided := true
	txCount, err := q.CountTransactionsBy(ctx, sqlc.CountTransactionsByParams{
		MemberID:      pgtype.UUID{Bytes: id, Valid: true},
		IncludeVoided: &includeVoided,
	})
	if err != nil {
		return mw.InternalError(c, "delete member.check_tx: ", err)
	}
	if txCount > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "该会员有交易记录，不能删除，建议改为停用")
	}

	if err := q.DeleteMember(ctx, id); err != nil {
		return mw.InternalError(c, "delete member: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.Member) MemberDTO {
	dto := MemberDTO{
		ID:           m.ID,
		Name:         m.Name,
		Phone:        m.Phone,
		Gender:       m.Gender,
		Notes:        m.Notes,
		Status:       m.Status,
		TotalBalance: "0",
		TotalPending: "0",
		CreatedAt:    m.CreatedAt.Time,
		UpdatedAt:    m.UpdatedAt.Time,
	}
	if m.Birthday.Valid {
		s := m.Birthday.Time.Format("2006-01-02")
		dto.Birthday = &s
	}
	return dto
}

func toListDTO(r sqlc.ListMembersRow) MemberDTO {
	dto := MemberDTO{
		ID:              r.ID,
		Name:            r.Name,
		Phone:           r.Phone,
		Gender:          r.Gender,
		Notes:           r.Notes,
		Status:          r.Status,
		TotalBalance:    r.TotalBalance.String(),
		TotalPending:    r.TotalPending.String(),
		CardCount:       r.CardCount,
		ActiveCardCount: r.ActiveCardCount,
		CreatedAt:       r.CreatedAt.Time,
		UpdatedAt:       r.UpdatedAt.Time,
	}
	if r.Birthday.Valid {
		s := r.Birthday.Time.Format("2006-01-02")
		dto.Birthday = &s
	}
	return dto
}


// checkPhoneUnique
//   - 占位号 '00000000000' 跳过校验（业务允许重复，如"旅行社"）
//   - excludeID：编辑时传当前会员 ID，排除自己
func checkPhoneUnique(c *echo.Context, q *sqlc.Queries, phone string, excludeID *uuid.UUID) error {
	if phone == "00000000000" {
		return nil
	}
	var ex pgtype.UUID
	if excludeID != nil {
		ex = pgtype.UUID{Bytes: *excludeID, Valid: true}
	}
	exists, err := q.ExistsMemberByPhone(c.Request().Context(), sqlc.ExistsMemberByPhoneParams{
		Phone:     phone,
		ExcludeID: ex,
	})
	if err != nil {
		return mw.InternalError(c, "members.check_phone", err)
	}
	if exists {
		return echo.NewHTTPError(http.StatusBadRequest, "该手机号已被其他会员使用")
	}
	return nil
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
