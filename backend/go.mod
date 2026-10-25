module github.com/zhaojiannet/mms/backend

go 1.26.0

// 依赖版本由容器内首次 go mod tidy 解析并写入 go.sum
// 禁止在宿主机执行 go 命令（见 CLAUDE.md 规则 9）

require (
	github.com/alexedwards/argon2id v1.0.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx-shopspring-decimal v0.0.0-20220624020537-1d36b5a1853e
	github.com/jackc/pgx/v5 v5.10.0
	github.com/labstack/echo/v5 v5.4.1
	github.com/mojocn/base64Captcha v1.3.8
	github.com/pressly/goose/v3 v3.27.3
	github.com/shopspring/decimal v1.4.0
	golang.org/x/crypto v0.56.0
	golang.org/x/image v0.45.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/air-verse/air v1.67.4 // indirect
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/bep/godartsass/v2 v2.5.0 // indirect
	github.com/bep/golibsass v1.2.0 // indirect
	github.com/bits-and-blooms/bitset v1.24.5 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gohugoio/hashstructure v0.6.0 // indirect
	github.com/gohugoio/hugo v0.164.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tdewolff/parse/v2 v2.8.12 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

tool github.com/air-verse/air
