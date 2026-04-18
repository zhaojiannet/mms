package cards

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID                   uuid.UUID       `json:"id"`
	MemberID             uuid.UUID       `json:"member_id"`
	CardTypeID           uuid.UUID       `json:"card_type_id"`
	CardTypeName         string          `json:"card_type_name"`
	CardTypePrice        decimal.Decimal `json:"card_type_price"`
	CardTypeDiscountRate decimal.Decimal `json:"card_type_discount_rate"`
	FinalPrice           decimal.Decimal `json:"final_price"`
	FinalDiscountRate    decimal.Decimal `json:"final_discount_rate"`
	Balance              decimal.Decimal `json:"balance"`
	IsCustom             bool            `json:"is_custom"` // final 与模板不同即视为定制
	IssuedAt             time.Time       `json:"issued_at"`
	ExpiresAt            *time.Time      `json:"expires_at"`
	Status               string          `json:"status"`
	Notes                *string         `json:"notes"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type IssueRequest struct {
	CardTypeID        uuid.UUID        `json:"card_type_id"`
	FinalPrice        *decimal.Decimal `json:"final_price"`         // 可选，不传则取 card_type.price
	FinalDiscountRate *decimal.Decimal `json:"final_discount_rate"` // 可选，不传则取 card_type.discount_rate
	Balance           *decimal.Decimal `json:"balance"`             // 可选，不传则 = final_price
	ExpiresAt         *string          `json:"expires_at"`          // ISO timestamp
	Notes             *string          `json:"notes"`
}

type UpdateRequest struct {
	FinalPrice        *decimal.Decimal `json:"final_price"`
	FinalDiscountRate *decimal.Decimal `json:"final_discount_rate"`
	Status            *string          `json:"status"`
	ExpiresAt         *string          `json:"expires_at"`
	Notes             *string          `json:"notes"`
}

// List GET /api/members/:memberId/cards
func ListByMember(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	rows, err := q.ListCardsByMember(c.Request().Context(), memberID)
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}

	items := make([]DTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, fromListRow(r))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get GET /api/cards/:id
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	r, err := q.GetCardByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	return c.JSON(http.StatusOK, fromGetRow(r))
}

// Issue POST /api/members/:memberId/cards
// 最小单元：只创建卡，不产生办卡 transaction（后续 A3 批次在 transactions 模块联动）
func Issue(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	var req IssueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.CardTypeID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "card_type_id required")
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	ctx := c.Request().Context()

	ct, err := q.GetCardTypeByID(ctx, req.CardTypeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card_type not found")
		}
		return mw.InternalError(c, "get card_type: ", err)
	}

	finalPrice := ct.Price
	if req.FinalPrice != nil {
		finalPrice = *req.FinalPrice
	}
	finalRate := ct.DiscountRate
	if req.FinalDiscountRate != nil {
		if req.FinalDiscountRate.LessThanOrEqual(decimal.Zero) || req.FinalDiscountRate.GreaterThan(decimal.NewFromInt(1)) {
			return echo.NewHTTPError(http.StatusBadRequest, "final_discount_rate must be in (0, 1]")
		}
		finalRate = *req.FinalDiscountRate
	}

	var balanceNull decimal.NullDecimal
	if req.Balance != nil {
		balanceNull = decimal.NullDecimal{Decimal: *req.Balance, Valid: true}
	}
	expiresAt, err := parseTimestamp(req.ExpiresAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid expires_at: "+err.Error())
	}

	t := mw.TenantFrom(c)
	m, err := q.IssueCard(ctx, sqlc.IssueCardParams{
		TenantID:          t.ID,
		MemberID:          memberID,
		CardTypeID:        req.CardTypeID,
		FinalPrice:        finalPrice,
		FinalDiscountRate: finalRate,
		Balance:           balanceNull,
		ExpiresAt:         expiresAt,
		Notes:             req.Notes,
	})
	if err != nil {
		return mw.InternalError(c, "issue card: ", err)
	}

	return c.JSON(http.StatusCreated, fromCard(m, ct))
}

// Update PUT /api/cards/:id
func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	var priceNull, rateNull decimal.NullDecimal
	if req.FinalPrice != nil {
		priceNull = decimal.NullDecimal{Decimal: *req.FinalPrice, Valid: true}
	}
	if req.FinalDiscountRate != nil {
		rateNull = decimal.NullDecimal{Decimal: *req.FinalDiscountRate, Valid: true}
	}
	expiresAt, err := parseTimestamp(req.ExpiresAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid expires_at: "+err.Error())
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdateCard(c.Request().Context(), sqlc.UpdateCardParams{
		ID:                id,
		FinalPrice:        priceNull,
		FinalDiscountRate: rateNull,
		Status:            req.Status,
		ExpiresAt:         expiresAt,
		Notes:             req.Notes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card not found")
		}
		return mw.InternalError(c, "update: ", err)
	}
	// 回查 card_type 拼 DTO
	ct, _ := q.GetCardTypeByID(c.Request().Context(), m.CardTypeID)
	return c.JSON(http.StatusOK, fromCard(m, ct))
}

// Delete DELETE /api/cards/:id
//   - 前置：无交易引用
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	usage, err := q.CountCardUsage(c.Request().Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return mw.InternalError(c, "check usage: ", err)
	}
	if usage > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "该卡已有交易引用，不能删除，建议冻结（status=frozen）")
	}
	if err := q.DeleteCard(c.Request().Context(), id); err != nil {
		return mw.InternalError(c, "delete: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// --- DTO 构造 ---

func fromListRow(r sqlc.ListCardsByMemberRow) DTO {
	return DTO{
		ID:                   r.ID,
		MemberID:             r.MemberID,
		CardTypeID:           r.CardTypeID,
		CardTypeName:         r.CardTypeName,
		CardTypePrice:        r.CardTypePrice,
		CardTypeDiscountRate: r.CardTypeDiscountRate,
		FinalPrice:           r.FinalPrice,
		FinalDiscountRate:    r.FinalDiscountRate,
		Balance:              r.Balance,
		IsCustom:             !r.FinalPrice.Equal(r.CardTypePrice) || !r.FinalDiscountRate.Equal(r.CardTypeDiscountRate),
		IssuedAt:             r.IssuedAt.Time,
		ExpiresAt:            tsPtr(r.ExpiresAt),
		Status:               r.Status,
		Notes:                r.Notes,
		CreatedAt:            r.CreatedAt.Time,
		UpdatedAt:            r.UpdatedAt.Time,
	}
}

func fromGetRow(r sqlc.GetCardByIDRow) DTO {
	return DTO{
		ID:                   r.ID,
		MemberID:             r.MemberID,
		CardTypeID:           r.CardTypeID,
		CardTypeName:         r.CardTypeName,
		CardTypeDiscountRate: r.CardTypeDiscountRate,
		FinalPrice:           r.FinalPrice,
		FinalDiscountRate:    r.FinalDiscountRate,
		Balance:              r.Balance,
		IssuedAt:             r.IssuedAt.Time,
		ExpiresAt:            tsPtr(r.ExpiresAt),
		Status:               r.Status,
		Notes:                r.Notes,
		CreatedAt:            r.CreatedAt.Time,
		UpdatedAt:            r.UpdatedAt.Time,
	}
}

func fromCard(c sqlc.Card, ct sqlc.CardType) DTO {
	return DTO{
		ID:                   c.ID,
		MemberID:             c.MemberID,
		CardTypeID:           c.CardTypeID,
		CardTypeName:         ct.Name,
		CardTypePrice:        ct.Price,
		CardTypeDiscountRate: ct.DiscountRate,
		FinalPrice:           c.FinalPrice,
		FinalDiscountRate:    c.FinalDiscountRate,
		Balance:              c.Balance,
		IsCustom:             !c.FinalPrice.Equal(ct.Price) || !c.FinalDiscountRate.Equal(ct.DiscountRate),
		IssuedAt:             c.IssuedAt.Time,
		ExpiresAt:            tsPtr(c.ExpiresAt),
		Status:               c.Status,
		Notes:                c.Notes,
		CreatedAt:            c.CreatedAt.Time,
		UpdatedAt:            c.UpdatedAt.Time,
	}
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func parseTimestamp(s *string) (pgtype.Timestamptz, error) {
	if s == nil || *s == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}
