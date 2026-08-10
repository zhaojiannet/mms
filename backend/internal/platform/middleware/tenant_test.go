package middleware

import "testing"

func TestBuildReservedSubdomains(t *testing.T) {
	// platformHosts 的 exact 集合里既有完整主机名，也有 localhost / 127.0.0.1 这种无点或多点的
	reserved := buildReservedSubdomains(map[string]struct{}{
		"backoffice.vip.example.com": {},
		"console.vip.example.com":    {},
		"localhost":                  {},
		"127.0.0.1":                  {},
	})

	for _, s := range []string{"backoffice", "console", "admin", "www", "api"} {
		if _, ok := reserved[s]; !ok {
			t.Errorf("%q 应被保留", s)
		}
	}
	// 无点的主机名没有首段可取，不该凭空塞进保留表
	if _, ok := reserved["localhost"]; !ok {
		t.Error("localhost 属于基础保留字，应仍在表内")
	}
	// 商户可用的普通 slug 不受影响
	if _, ok := reserved["mystore"]; ok {
		t.Error("mystore 不该被保留")
	}
}
