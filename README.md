# 通用会员管理系统 · SaaS 版

**简单高效，管理不用愁，经营更省心**

面向小微商户的多租户 SaaS 会员管理平台。代码 AGPL-3.0 开源。

适用于：**美业、美容、美发、美甲、按摩、瑜伽、培训、宠物**等行业。

## 预览

![登录页](./docs/screenshots/01-login.png)

登录页：图形验证码 + 行业主题背景（美业 / 瑜伽 / 宠物等多套预设）。

![POS 收银工作区](./docs/screenshots/02-pos.png)

POS 收银工作区：选项目 / 多卡组合扣费 / 价格调整 / 一键结算。

![系统设置](./docs/screenshots/03-settings.png)

系统设置：店铺品牌、服务项目、卡型、员工、支付方式、撤单授权集中管理。

## 功能

| 模块 | 功能 |
|------|------|
| 收银台 | 快速结算、多支付方式、会员卡支付、智能多卡组合、价格调整 |
| 会员管理 | 会员档案、办卡充值、余额查询、挂账管理、消费记录 |
| 预约管理 | 顾客端在线预约（扫码直达）、状态追踪、预约码一键重置 |
| 营业报表 | 营业概览、支付统计、项目排行、会员卡销售、生日提醒、沉睡会员 |
| 系统设置 | 店铺品牌 / 服务项目 / 卡类型 / 员工 / 支付方式 / 账号 / 交易撤销 / 预约配置 |
| 操作日志 | 所有写操作追加审计，append-only 触发器拒绝 UPDATE/DELETE |
| 多租户 | 每个商户独立数据 + 独立子域名 + 独立品牌（PostgreSQL FORCE RLS 隔离） |
| 多店员 + 权限 | 超级管理员 / 管理员 / 员工 三级权限 + JWT token 版本吊销 |
| 通知与公告 | 顶部铃铛实时提醒（生日 / 预约待确认 / 系统公告）+ 侧边栏版本号 + changelog 页 |

**计费订阅、行业模块、运营后台** 规划中，当前版本（V 26.4）未实现。

## 技术栈

- 前端：Nuxt 4 + Nuxt UI v4
- 后端：Go 1.26 + Echo v5
- 数据库：PostgreSQL 18（行级安全 RLS 实现多租户隔离）
- 查询：sqlc + pgx/v5 + shopspring/decimal（金额全链路类型安全）
- 迁移：goose
- 部署：Docker Compose + 1Panel（OpenResty 反代 + 证书管理）

## 三种部署模式

| 模式 | 适用 | 说明 |
|---|---|---|
| **官方托管**（`hosted`） | 普通商户 | 官方运营的托管服务（自助注册与计费订阅在阶段 4+ 开放） |
| **社区自建**（`self-hosted`） | 技术用户 / 个人店面 | 开源免费，自己搭服务器 |
| **企业私有部署**（`enterprise`） | 连锁品牌 / 合规客户 | 合同定制，支持定制开发和 SLA |

通过环境变量 `DEPLOYMENT_MODE` 切换。

## 快速开始（自建）

### 1. 克隆项目

```bash
git clone https://github.com/zhaojiannet/mms.git
cd mms
```

### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 填写密码和密钥（参考注释）。生成密钥：`openssl rand -hex 64`

### 3. 启动服务

```bash
docker compose up -d
```

### 4. 访问

- 后端 health：http://localhost:8081/health（宿主 8081 → 容器 8080）
- 前端：http://localhost:3001（宿主 3001 → 容器 3000）
- 管理员账号：`.env` 的 `BOOTSTRAP_ADMIN_EMAIL` + `BOOTSTRAP_ADMIN_PASSWORD`（首次启动自动创建）

## 套餐（仅官方托管）

| 套餐 | 月付 | 年付 | 会员 | 员工 | 门店 | 微信推送/月 |
|---|---|---|---|---|---|---|
| **免费版** | ¥0 | — | 100 | 2 | 1 | 100 |
| **Plus** | ¥9.9 | ¥99 | 1000 | 10 | 1 | 1000 |
| **Pro** | ¥39.9 | ¥399 | 5000 | 50 | 3 | 5000 |
| **Ultra** | ¥199.9 | ¥1999 | 无限 | 无限 | 无限 | 无限 |
| **企业版** | 合同 | — | 私有部署 + 定制开发 + SLA | | | |

所有套餐均解锁**全部功能**，差异仅在数量与推送配额（以及 Ultra 的行业模块包）。

自建版**无套餐限制**（所有配额项视为无限）。

## 多租户 / 商户子域访问

每个商户访问自己的专属子域：

```
<slug>.vip.zhaojian.net         如 mystore.vip.zhaojian.net
```

开发环境：`<slug>.dev.example.com`。

数据隔离基于 PostgreSQL Row Level Security：数据库强制执行按 `tenant_id` 过滤，即使代码写错也不会跨租户泄露。

## 品牌定制（每个商户独立）

商户在"设置 → 店铺"里配置：
- 店铺名称、Logo（上传 PNG/JPG/WebP，启动时剥离 polyglot payload）
- 登录背景主题（美业 / 美容 / 美发 / 美甲 / 按摩 / 瑜伽 / 培训 / 宠物 等预设）

## 微信 / 短信通知推送

**当前版本未实现**，规划中。已有"顶部铃铛 + 站内通知"作为替代。

## 生产部署（1Panel）

推荐用 [1Panel](https://1panel.cn/) 管理。

### 服务器初始化（一次性）

1. 1Panel 应用商店装 OpenResty
2. 在 1Panel "网站" 给**每个商户**建一个反代站点（`<slug>.vip.zhaojian.net`，DNS 手动加 A 记录），站点配置内按路径分流，后端按 Host 子域解析租户：

   ```nginx
   location /api/     { proxy_pass http://127.0.0.1:8081; }   # Go 后端
   location /uploads/ { proxy_pass http://127.0.0.1:8081; }   # 店铺 logo 等上传资产
   location = /health { proxy_pass http://127.0.0.1:8081; access_log off; }
   location /         { proxy_pass http://127.0.0.1:3001; }   # Nuxt 前端
   ```

3. 每个站点单独申请 Let's Encrypt 证书（HTTP 验证，自动续期）。有意不用通配符证书：通配须 DNS 验证，等于把域名解析的 API 密钥存进服务器；单域手动建站每商户约 5 分钟，规模上来（约 30 商户/月）再考虑自动化
4. clone 仓库到服务器（如 `/opt/mms`），`cp .env.example .env` 填好配置，然后生产模式启动（PG 复用全局 `postgres-server`）：

   ```bash
   mkdir -p frontend/.output   # 前端产物占位，首次推送前容器起不来属正常
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
   ```

5. 备份定时任务：`crontab -e` 加 `0 3 * * * /opt/mms/ops/backup_pg.sh >> /var/log/mms_backup.log 2>&1`

### 日常发布

在**本机**执行（首次先 `cp ops/deploy.env.example ops/deploy.env` 填服务器地址）：

```bash
./ops/deploy.sh            # 全量（前端 + 后端）
./ops/deploy.sh frontend   # 只改了前端
./ops/deploy.sh backend    # 只改了后端
```

脚本自动完成：本机容器内构建（前端 `nuxi build`、后端编译检查）→ 服务器留回滚快照（后端代码 tar + `pg_dump`）→ rsync 推送（永不触碰服务器 `.env` 和商户上传文件）→ 重启（后端启动时 goose 自动迁移）→ 轮询 `/health` 健康检查。失败时打印回滚命令。

前端回滚不靠快照：本机 checkout 上一个正常 commit 重新 `./ops/deploy.sh frontend` 即可。

## 开发

```bash
docker compose up              # 启动
docker compose logs -f backend # 查看日志

# 进入后端容器
docker compose exec backend sh
go mod tidy                    # 刷新依赖
go run ./cmd/server            # 手动启动

# 进入数据库
docker exec -it postgres-server psql -U mms -d mms
```

## 项目结构

```
mms/
├── backend/                      Go 后端
│   ├── cmd/server/main.go        入口
│   ├── internal/
│   │   ├── core/                 业务模块
│   │   │   ├── auth/             登录 / captcha / 账号锁定
│   │   │   ├── members/          会员 CRUD
│   │   │   ├── cards/            会员卡
│   │   │   ├── card_types/       卡型
│   │   │   ├── transactions/     消费 / 办卡 / 清账 / 撤销（FOR UPDATE 锁）
│   │   │   ├── member_credits/   挂账
│   │   │   ├── appointments/     预约
│   │   │   ├── booking/          对外预约页 API（无鉴权）
│   │   │   ├── staff/            员工
│   │   │   ├── services/         服务项目
│   │   │   ├── payment_methods/  支付方式
│   │   │   ├── reports/          报表
│   │   │   ├── tenant_settings/  店铺名 / Logo / 登录背景 / 撤单开关
│   │   │   ├── users/            账号管理（super_admin 专属）
│   │   │   ├── audit_logs/       操作日志
│   │   │   ├── notifications/    通知与系统公告（含 announcements.json seed）
│   │   │   └── uploads/          Logo 上传（重编码剥离 polyglot）
│   │   └── platform/             基础设施
│   │       ├── db/               pgxpool + DSN 构造
│   │       ├── auth/             JWT 签发 / 解析 / 版本吊销
│   │       ├── middleware/       TenantResolver / TenantTx / RequireAuth / Audit
│   │       └── bootstrap/        首次启动创建超管
│   ├── migrations/               goose SQL 迁移（00001-00030）
│   ├── sqlc/                     sqlc 生成代码
│   ├── sqlc.yaml
│   └── Dockerfile
├── frontend/                     Nuxt 4 前端（SPA 模式）
│   ├── app/
│   │   ├── pages/                收银 / 会员 / 预约 / 报表 / 设置 / changelog
│   │   ├── components/           SidebarContent / UserMenu / PosWorkbench / EmptyState
│   │   ├── composables/          useApi / useStoreInfo / useTheme / useSafeUrl
│   │   ├── middleware/           auth.global / super-admin / at-least-admin
│   │   └── stores/               auth (Pinia)
│   ├── nuxt.config.ts
│   └── Dockerfile
├── ops/                          运维脚本
├── docs/                         设计文档
│   ├── DESIGN.md
│   ├── decisions.md              关键决策记录
│   └── RELEASING.md              发版流程
├── docker-compose.yml
├── .env.example
└── README.md
```

> 阶段 4+ 按需添加：计费订阅 / 自助注册 / 支付 / 运营后台 / 行业扩展。当前不预留空目录。

## 许可证

GNU Affero General Public License v3.0 — 见 [LICENSE](./LICENSE)

任何基于本代码的修改，若通过网络向用户提供服务，必须按同样许可证开源修改后的代码。

## 版权

© 2026 赵健
