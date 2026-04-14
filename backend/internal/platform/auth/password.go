package auth

import "github.com/alexedwards/argon2id"

// HashPassword 使用 argon2id 默认参数（RFC 9106 推荐集）生成密码哈希
// 返回值格式：$argon2id$v=19$m=65536,t=1,p=2$...$...，自带 salt + 参数
func HashPassword(plain string) (string, error) {
	return argon2id.CreateHash(plain, argon2id.DefaultParams)
}

// VerifyPassword 常量时间比较 plain 与已存储的 hash
func VerifyPassword(plain, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(plain, hash)
}
