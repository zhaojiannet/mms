# 通用会员管理系统 · SaaS 版

**简单高效，管理不用愁，经营更省心**

面向小微商户的多租户 SaaS 会员管理平台。代码 AGPL-3.0 开源。

适用于：**美业、美容、美发、美甲、按摩、瑜伽、培训、宠物**等行业。

## 功能

| 模块 | 功能 |
|------|------|
| 收银台 | 快速结算、多支付方式、会员卡支付、智能多卡组合、价格调整 |
| 会员管理 | 会员档案、办卡充值、余额查询、挂账管理、消费记录 |
| 预约管理 | 用户端在线预约、状态追踪、微信通知推送 |
| 营业报表 | 营业概览、支付统计、项目排行、生日提醒、沉睡会员 |
| 系统设置 | 服务项目、卡类型、员工管理、交易撤销 |
| **多租户**（新） | 每个商户独立数据 + 独立子域名 + 独立品牌 |
| **计费订阅**（新） | 五档套餐 + 微信/支付宝支付 + 自助升级 |
| **行业模块**（新） | 美业/餐饮/健身等按需开通 |
| **运营后台**（新） | 超管可查看/管理所有商户 |

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
| **官方托管**（`hosted`） | 普通商户 | 访问 https://demo.example.com 自助注册，按套餐付费 |
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

- 后端：http://localhost:8080/health
- 前端：http://localhost:3000
- 管理员账号：见 `.env` 的 `BOOTSTRAP_ADMIN_*`

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
demo.demo.example.com         示例商户
<slug>.demo.example.com       其他商户
```

数据隔离基于 PostgreSQL Row Level Security：数据库强制执行按 `tenant_id` 过滤，即使代码写错也不会跨租户泄露。

## 品牌定制（每个商户独立）

商户在各自的系统设置里配置：
- 店铺名称、Logo、主题色
- 行业类型（决定默认背景图：美业、美容、美发、美甲、按摩、瑜伽、培训、宠物）
- 自定义字段（扩展会员档案、预约单等）

## 微信通知推送

支持通过微信公众号推送预约通知。

### 配置步骤

1. 申请微信公众号测试号：https://mp.weixin.qq.com/debug/cgi-bin/sandboxinfo

2. 创建模板消息，内容如下：
```
姓名：{{name.DATA}}
电话：{{phone.DATA}}
时间：{{time.DATA}}
项目：{{services.DATA}}
员工：{{staff.DATA}}
留言：{{message.DATA}}
```

> 注意：不要使用 `notes` 或 `remark` 作为字段名，这是微信保留字段，不会显示。

3. 部署 Cloudflare Worker（文件在 `cloudflare-workers/` 目录）

4. 配置环境变量：
```env
WXPUSH_URL=https://your-worker.workers.dev/wxsend
WXPUSH_TOKEN=your-api-token
```

## 生产部署（1Panel）

推荐用 [1Panel](https://1panel.cn/) 管理：

1. 1Panel 应用商店装 OpenResty
2. 在 1Panel "网站" 新建反向代理站点：
   - 域名：`demo.example.com` → 反代到 `127.0.0.1:3000`（Nuxt）
   - 域名：`<slug>.demo.example.com` → 反代到 `127.0.0.1:3000`（Nuxt，前端按 Host 分租户）
3. 1Panel "证书" 申请 Let's Encrypt，自动续期
4. 1Panel "容器"里跑 docker-compose（后端 + 前端 + PG）

## 开发

```bash
docker compose up              # 启动
docker compose logs -f backend # 查看日志

# 进入后端容器
docker compose exec backend sh
go mod tidy                    # 刷新依赖
go run ./cmd/server            # 手动启动

# 进入数据库
docker compose exec postgres psql -U mms -d mms
```

## 项目结构

```
mms/
├── backend/                   Go 后端
│   ├── cmd/server/main.go     入口
│   ├── internal/              业务模块
│   │   ├── core/              tenant/user/role/feature
│   │   ├── members/           会员
│   │   ├── cards/             会员卡
│   │   ├── transactions/      收银
│   │   ├── appointments/      预约
│   │   ├── reports/           报表
│   │   ├── notifications/     通知
│   │   ├── billing/           订阅/套餐（hosted 模式）
│   │   ├── signup/            自助注册（hosted 模式）
│   │   ├── payment/           微信/支付宝（hosted 模式）
│   │   ├── admin/             运营后台
│   │   └── industries/        行业模块（美业/餐饮等）
│   ├── migrations/            goose SQL 迁移
│   ├── sqlc/                  sqlc 生成代码
│   ├── sqlc.yaml
│   └── Dockerfile
├── frontend/                  Nuxt 4 前端
│   ├── app/
│   ├── nuxt.config.ts
│   └── Dockerfile
├── cloudflare-workers/        微信推送 Workers
├── docker-compose.yml
├── .env.example
└── README.md
```

## 许可证

GNU Affero General Public License v3.0 — 见 [LICENSE](./LICENSE)

任何基于本代码的修改，若通过网络向用户提供服务，必须按同样许可证开源修改后的代码。

## 版权

© 2026 赵健
