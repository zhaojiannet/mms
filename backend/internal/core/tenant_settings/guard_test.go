package tenantsettings

import "testing"

// TestSecretKeysNeverReadable 凭据类 key 不得经通用 KV 端点读出。
// 回归背景：super_password_hash 与普通配置同表，而 List/Get 对 staff 开放，
// 曾可被任意员工读到超管密码哈希后离线爆破，进而绕过撤单授权。
func TestSecretKeysNeverReadable(t *testing.T) {
	secrets := []string{
		"super_password_hash",
		"any_future_credential_hash", // 后缀规则：新增同类 key 自动受保护
	}
	for _, k := range secrets {
		if !isSecret(k) {
			t.Errorf("isSecret(%q) 应为 true——凭据类 key 必须对通用读端点不可见", k)
		}
		if !isWriteGuarded(k) {
			t.Errorf("isWriteGuarded(%q) 应为 true——凭据类 key 也不允许通用写", k)
		}
	}
}

// TestGuardedKeysNotWritable 受专用端点管辖的状态项可读但不可经通用端点改，
// 否则 admin 能绕过 super_admin 专属校验篡改撤单授权状态或预约码。
func TestGuardedKeysNotWritable(t *testing.T) {
	guarded := []string{
		"enable_transaction_void",
		"void_enabled_at",
		"void_enabled_by",
		"booking_code",
		"booking_code_updated_at",
	}
	for _, k := range guarded {
		if !isWriteGuarded(k) {
			t.Errorf("isWriteGuarded(%q) 应为 true", k)
		}
		if isSecret(k) {
			t.Errorf("isSecret(%q) 应为 false——这些项前端需要展示", k)
		}
	}
}

// TestOrdinarySettingsUnaffected 普通配置项照常可读可写，闸门不能误伤。
func TestOrdinarySettingsUnaffected(t *testing.T) {
	ordinary := []string{"store_name", "login_bg_theme", "store_logo_url", "enable_login_captcha"}
	for _, k := range ordinary {
		if isSecret(k) {
			t.Errorf("isSecret(%q) 应为 false", k)
		}
		if isWriteGuarded(k) {
			t.Errorf("isWriteGuarded(%q) 应为 false——普通配置项须保持可写", k)
		}
	}
}
