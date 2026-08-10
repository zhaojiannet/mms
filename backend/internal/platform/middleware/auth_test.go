package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/sqlc"
)

func TestRequirePasswordChanged(t *testing.T) {
	run := func(mustChange bool) (reached bool, status int) {
		e := echo.New()
		c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
		c.Set(string(CtxKeyUser), sqlc.User{MustChangePassword: mustChange})

		err := RequirePasswordChanged()(func(*echo.Context) error {
			reached = true
			return nil
		})(c)

		if err == nil {
			return reached, http.StatusOK
		}
		he, ok := err.(*echo.HTTPError)
		if !ok {
			t.Fatalf("期望 *echo.HTTPError，得到 %T", err)
		}
		return reached, he.Code
	}

	if reached, status := run(true); reached || status != http.StatusForbidden {
		t.Errorf("待改密用户应被 403 挡下，得到 reached=%v status=%d", reached, status)
	}
	if reached, status := run(false); !reached || status != http.StatusOK {
		t.Errorf("已改密用户应放行，得到 reached=%v status=%d", reached, status)
	}
}
