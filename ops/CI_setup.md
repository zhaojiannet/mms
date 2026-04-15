# CI 集成指南

项目本身不强绑定 CI 平台。下面是推荐的 GitHub Actions / Gitea Actions 配置，复制到 `.github/workflows/`（GitHub）或 `.gitea/workflows/`（Gitea）即可生效。

## 1. 密钥扫描（gitleaks）

`.github/workflows/gitleaks.yml`：

```yaml
name: gitleaks
on:
  push:
    branches: [main, master]
  pull_request:
  schedule:
    - cron: '0 3 * * 1'

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: gitleaks/gitleaks-action@v2
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITLEAKS_CONFIG: .gitleaks.toml
```

本地手动扫描：`./ops/gitleaks_scan.sh`

## 2. Go 测试（建议）

```yaml
name: backend-test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:18
        env:
          POSTGRES_PASSWORD: postgres
        ports: ['5432:5432']
        options: --health-cmd pg_isready
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - working-directory: backend
        run: |
          go vet ./...
          go test ./...
```

## 3. 前端 typecheck

```yaml
name: frontend-check
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v3
      - uses: actions/setup-node@v4
        with: { node-version: '20', cache: pnpm }
      - working-directory: frontend
        run: |
          pnpm install --frozen-lockfile
          pnpm typecheck
```

---

## Pre-commit hook（本地）

```bash
# 在仓库根目录跑：
cat > .git/hooks/pre-commit <<'EOF'
#!/bin/sh
exec ./ops/gitleaks_scan.sh --no-git
EOF
chmod +x .git/hooks/pre-commit
```
