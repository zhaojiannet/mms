package auth

import (
	"strings"

	"github.com/alexedwards/argon2id"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 argon2id 默认参数（RFC 9106 推荐集）生成密码哈希
// 返回值格式：$argon2id$v=19$m=65536,t=1,p=2$...$...，自带 salt + 参数
func HashPassword(plain string) (string, error) {
	return argon2id.CreateHash(plain, argon2id.DefaultParams)
}

// VerifyPassword 兼容两种哈希算法：
//   - argon2id（新系统默认，$argon2id$... 前缀）
//   - bcrypt（兼容老 demo 迁入的用户，$2a$/$2b$/$2y$ 前缀，Node.js bcryptjs 生成）
//
// 常量时间比较。新写入始终用 HashPassword → argon2id。
func VerifyPassword(plain, hash string) (bool, error) {
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
		if err == nil {
			return true, nil
		}
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	}
	return argon2id.ComparePasswordAndHash(plain, hash)
}
