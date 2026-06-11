package cards

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// 卡的写操作只保留 transactions.IssueCard（办卡+收款+流水联动）一个入口。
// 曾经的裸发卡 POST /members/:id/cards 不建交易、不写流水、金额无下限，
// 是绕过对账不变量「balance = SUM(delta)」的造钱通道，已整体移除；
// GET/PUT/DELETE /cards/:id 无前端调用，一并移除（删卡还会级联销毁资金流水）。

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

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
