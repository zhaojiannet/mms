// Package reports 提供 8 个统计接口（老 demo reports.js 对等实现，修 quantity 未累加的 bug）
package reports

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/echox"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// ================= 1. 营业报表 =================

type BusinessResponse struct {
	SaleRevenue     decimal.Decimal `json:"sale_revenue"`      // 服务消费实收（不含办卡）
	CreditCharged   decimal.Decimal `json:"credit_charged"`    // 新增挂账
	TotalRevenue    decimal.Decimal `json:"total_revenue"`     // sale_revenue + credit_charged（老"总消费"口径）
	CardConsumption decimal.Decimal `json:"card_consumption"`  // 卡耗
	RechargeAmount  decimal.Decimal `json:"recharge_amount"`   // 充值
	CustomerCount   int64           `json:"customer_count"`
	AverageOrder    decimal.Decimal `json:"average_order_value"`
}

func Business(c *echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return err
	}
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()

	row, err := q.ReportBusiness(ctx, sqlc.ReportBusinessParams{
		TransactionTime: start, TransactionTime_2: end,
	})
	if err != nil {
		return mw.InternalError(c, "business: ", err)
	}
	credit, err := q.ReportCreditsCharged(ctx, sqlc.ReportCreditsChargedParams{
		ChargedAt: start, ChargedAt_2: end,
	})
	if err != nil {
		return mw.InternalError(c, "credits: ", err)
	}

	total := row.SaleRevenue.Add(credit)
	avg := decimal.Zero
	if row.CustomerCount > 0 {
		avg = total.Div(decimal.NewFromInt(row.CustomerCount))
	}

	return c.JSON(http.StatusOK, BusinessResponse{
		SaleRevenue:     row.SaleRevenue,
		CreditCharged:   credit,
		TotalRevenue:    total,
		CardConsumption: row.CardConsumption,
		RechargeAmount:  row.RechargeAmount,
		CustomerCount:   row.CustomerCount,
		AverageOrder:    avg,
	})
}

// ================= 1.5 卡池余额 =================

// CardPool GET /api/reports/card-pool
// 返回当前店内所有 active 卡的余额总和（前端 KPI 用，避免拉全量会员算）
func CardPool(c *echo.Context) error {
	total, err := sqlc.New(mw.TxFrom(c)).ReportCardPool(c.Request().Context())
	if err != nil {
		return mw.InternalError(c, "reports.card_pool", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"total": total})
}

// ================= 2. 项目销售排行 =================

func ServiceRanking(c *echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return err
	}
	page, limit, offset := echox.Paging(c, 50, 200)

	rows, err := sqlc.New(mw.TxFrom(c)).ReportServiceRanking(c.Request().Context(), sqlc.ReportServiceRankingParams{
		TransactionTime: start, TransactionTime_2: end, Limit: limit, Offset: offset,
	})
	if err != nil {
		return mw.InternalError(c, "ranking: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"data":  rows,
		"page":  page,
		"limit": limit,
	})
}

// ================= 3. 沉睡会员（90 天无 sale）=================

func SleepingMembers(c *echo.Context) error {
	page, limit, offset := echox.Paging(c, 50, 200)
	cutoff := time.Now().AddDate(0, 0, -90)
	rows, err := sqlc.New(mw.TxFrom(c)).ReportSleepingMembers(c.Request().Context(), sqlc.ReportSleepingMembersParams{
		TransactionTime: pgtype.Timestamptz{Time: cutoff, Valid: true},
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		return mw.InternalError(c, "sleeping: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"data": rows, "page": page, "limit": limit})
}

// ================= 4. 会员消费排行 =================

func MemberRanking(c *echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return err
	}
	page, limit, offset := echox.Paging(c, 50, 200)
	rows, err := sqlc.New(mw.TxFrom(c)).ReportMemberRanking(c.Request().Context(), sqlc.ReportMemberRankingParams{
		TransactionTime: start, TransactionTime_2: end, Limit: limit, Offset: offset,
	})
	if err != nil {
		return mw.InternalError(c, "member rank: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"data": rows, "page": page, "limit": limit})
}

// ================= 5. 生日提醒（未来 15 天）=================

func BirthdayReminders(c *echo.Context) error {
	rows, err := sqlc.New(mw.TxFrom(c)).ReportBirthdayReminders(c.Request().Context())
	if err != nil {
		return mw.InternalError(c, "birthday: ", err)
	}
	if rows == nil {
		rows = []sqlc.ReportBirthdayRemindersRow{}
	}
	return c.JSON(http.StatusOK, map[string]any{"data": rows})
}

// ================= 6. 支付方式构成 =================

func PaymentSummary(c *echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return err
	}
	rows, err := sqlc.New(mw.TxFrom(c)).ReportPaymentSummary(c.Request().Context(), sqlc.ReportPaymentSummaryParams{
		TransactionTime: start, TransactionTime_2: end,
	})
	if err != nil {
		return mw.InternalError(c, "payment: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"data": rows})
}

// ================= 7. 卡销售分析 =================

func CardSalesSummary(c *echo.Context) error {
	start, end, err := parseDateRange(c)
	if err != nil {
		return err
	}
	rows, err := sqlc.New(mw.TxFrom(c)).ReportCardSalesSummary(c.Request().Context(), sqlc.ReportCardSalesSummaryParams{
		IssuedAt: start, IssuedAt_2: end,
	})
	if err != nil {
		return mw.InternalError(c, "card sales: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"data": rows})
}

// ================= 8. 挂账统计 =================

type PendingStatsResponse struct {
	Summary PendingStatsSummary              `json:"summary"`
	Data    []sqlc.ReportPendingByMemberRow  `json:"data"`
	Page    int32                            `json:"page"`
	Limit   int32                            `json:"limit"`
}

type PendingStatsSummary struct {
	RecordCount   int64           `json:"record_count"`
	MemberCount   int64           `json:"member_count"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	AverageAmount decimal.Decimal `json:"average_amount"`
}

func PendingStats(c *echo.Context) error {
	page, limit, offset := echox.Paging(c, 50, 200)
	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	stats, err := q.ReportPendingStats(ctx)
	if err != nil {
		return mw.InternalError(c, "stats: ", err)
	}
	rows, err := q.ReportPendingByMember(ctx, sqlc.ReportPendingByMemberParams{Limit: limit, Offset: offset})
	if err != nil {
		return mw.InternalError(c, "by member: ", err)
	}

	avg := decimal.Zero
	if stats.MemberCount > 0 {
		avg = stats.TotalAmount.Div(decimal.NewFromInt(stats.MemberCount))
	}

	return c.JSON(http.StatusOK, PendingStatsResponse{
		Summary: PendingStatsSummary{
			RecordCount:   stats.RecordCount,
			MemberCount:   stats.MemberCount,
			TotalAmount:   stats.TotalAmount,
			AverageAmount: avg,
		},
		Data: rows, Page: page, Limit: limit,
	})
}

// ================= 工具 =================

func parseDateRange(c *echo.Context) (pgtype.Timestamptz, pgtype.Timestamptz, error) {
	startStr := c.QueryParam("start_date")
	endStr := c.QueryParam("end_date")
	if startStr == "" || endStr == "" {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{},
			echo.NewHTTPError(http.StatusBadRequest, "start_date and end_date required (YYYY-MM-DD)")
	}
	// 业务日界用进程时区（容器 TZ 环境变量，默认 Asia/Shanghai）：
	// 按 UTC 解析会让 UTC+8 商户的"今日营收"偏移 8 小时
	start, err := time.ParseInLocation("2006-01-02", startStr, time.Local)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{},
			echo.NewHTTPError(http.StatusBadRequest, "invalid start_date")
	}
	end, err := time.ParseInLocation("2006-01-02", endStr, time.Local)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{},
			echo.NewHTTPError(http.StatusBadRequest, "invalid end_date")
	}
	// end 包含当日（使用下一天 00:00 做 [start, end) 左闭右开）
	end = end.Add(24 * time.Hour)
	return pgtype.Timestamptz{Time: start, Valid: true},
		pgtype.Timestamptz{Time: end, Valid: true}, nil
}

