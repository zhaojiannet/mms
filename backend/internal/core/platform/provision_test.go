package platform

import "testing"

func TestValidateSlug(t *testing.T) {
	valid := []string{"mystore", "my-first-shop", "abc", "a2b", "demo123"}
	for _, s := range valid {
		if err := ValidateSlug(s); err != nil {
			t.Errorf("ValidateSlug(%q) 应通过，得到: %v", s, err)
		}
	}
	invalid := []string{
		"ab",            // 太短
		"Ab-c",          // 大写
		"1abc",          // 数字开头
		"abc-",          // 连字符结尾
		"a_b_c",         // 下划线
		"api",           // 保留子域
		"admin",         // 保留子域
		"www",           // 保留子域
		"中文slug",        // 非 ASCII
		"a" + string(make([]byte, 40)), // 超长
	}
	for _, s := range invalid {
		if err := ValidateSlug(s); err == nil {
			t.Errorf("ValidateSlug(%q) 应拒绝", s)
		}
	}
}
