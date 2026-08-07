package auth

// 图形验证码：基于 mojocn/base64Captcha（纯本地，零外部依赖）
//
// 设计取舍：
//   - 使用进程内存 store（sync.Mutex + map），5 分钟 TTL
//   - 单实例部署足够；横向扩展需换成 Redis（共享 store）
//   - 触发时机：失败 >= 2 次后由前端主动请求，登录时带回 captcha_id/answer
//   - 后端不强制：缺失 captcha 参数不直接拒绝，但会让"被锁定用户无法自救"
//   - 用途 1：增加自动化攻击成本（密码喷射 / 凭证填充）
//   - 用途 2：解决"恶意锁号 DoS"—— 正确 captcha + 正确密码可绕过 locked_until

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/mojocn/base64Captcha"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

type captchaEntry struct {
	answer   string
	expireAt time.Time
}

var (
	captchaStoreData = make(map[string]captchaEntry)
	captchaStoreMu   sync.Mutex
)

const captchaTTL = 5 * time.Minute

// startCaptchaGC 后台 GC：每分钟清理过期 entry
// 不用 init()：避免导入即启线程（测试不友好）；改由 auth.Init 调用
func startCaptchaGC() {
	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		for range t.C {
			captchaStoreMu.Lock()
			now := time.Now()
			for k, v := range captchaStoreData {
				if now.After(v.expireAt) {
					delete(captchaStoreData, k)
				}
			}
			captchaStoreMu.Unlock()
		}
	}()
}

// captchaMemStore 实现 base64Captcha.Store 接口
type captchaMemStore struct{}

func (captchaMemStore) Set(id, value string) error {
	captchaStoreMu.Lock()
	captchaStoreData[id] = captchaEntry{answer: value, expireAt: time.Now().Add(captchaTTL)}
	captchaStoreMu.Unlock()
	return nil
}

func (captchaMemStore) Get(id string, clear bool) string {
	captchaStoreMu.Lock()
	defer captchaStoreMu.Unlock()
	e, ok := captchaStoreData[id]
	if !ok || time.Now().After(e.expireAt) {
		delete(captchaStoreData, id)
		return ""
	}
	if clear {
		delete(captchaStoreData, id)
	}
	return e.answer
}

func (s captchaMemStore) Verify(id, answer string, clear bool) bool {
	expected := s.Get(id, clear)
	return expected != "" && expected == answer
}

// driver：4 位数字，80×240，线噪 + 点噪
var captchaDriver = base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
var captchaInstance = base64Captcha.NewCaptcha(captchaDriver, captchaMemStore{})

// CaptchaResponse 返回给前端的格式
type CaptchaResponse struct {
	ID    string `json:"id"`    // 验证码 ID
	Image string `json:"image"` // data:image/png;base64,...
}

// CaptchaHandler GET /api/captcha
// 无需登录，但挂了 per-IP 速率限制（防刷）
func CaptchaHandler(c *echo.Context) error {
	id, b64, _, err := captchaInstance.Generate()
	if err != nil {
		return mw.InternalError(c, "captcha.generate", err)
	}
	return c.JSON(http.StatusOK, CaptchaResponse{ID: id, Image: b64})
}

// verifyCaptcha 校验并销毁（一次性）
// 返回 true 当且仅当 id 存在、未过期、answer 匹配
func verifyCaptcha(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	return captchaMemStore{}.Verify(id, answer, true)
}

// VerifyCaptcha 导出版：平台登录与公开申请表单共用同一 captcha store
func VerifyCaptcha(id, answer string) bool {
	return verifyCaptcha(id, answer)
}
