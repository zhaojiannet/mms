#!/usr/bin/env bash
# 本机一键部署到生产服务器：容器内构建 → rsync 推送 → 服务器重启 → 健康检查
#
# 用法（项目根目录）：
#   cp ops/deploy.env.example ops/deploy.env && vi ops/deploy.env   # 首次
#   ./ops/deploy.sh            # 全量（前端 + 后端）
#   ./ops/deploy.sh frontend   # 仅前端
#   ./ops/deploy.sh backend    # 仅后端
#
# 前提：服务器用 docker-compose.prod.yml 叠加启动（见该文件头部说明）
#
# 与 QingSi 版的差异（按 MMS 实际调整）：
#   - 后端本机交叉编译推二进制，服务器零外网依赖、重启秒起；goose 迁移嵌在
#     二进制里随启动自动执行，所以部署前先 pg_dump 快照，出问题可回滚
#   - 前端在本机容器内 nuxi build，产物经 bind mount 落在 frontend/.output 直接 rsync
#
# 安全边界：
#   - 服务器地址、域名等私域信息只在 ops/deploy.env（gitignore，不入库）
#   - rsync 永不触碰服务器上的 .env 和 uploads
set -euo pipefail
cd "$(dirname "$0")/.."

[ -f ops/deploy.env ] && . ops/deploy.env

# 服务器上 PG 容器名（1Panel 装的名字不固定），deploy.env 可覆盖
PG_CONTAINER="${PG_CONTAINER:-postgres-server}"

for v in SERVER APP_DIR SITE_URL BACKUP_DIR; do
	if [ -z "${!v:-}" ]; then
		echo "deploy.env 缺少 ${v}。先执行：cp ops/deploy.env.example ops/deploy.env 并按注释填写" >&2
		exit 1
	fi
done

PART="${1:-all}"

if [ "${PART}" != "backend" ]; then
	echo "== 前端：本机容器内构建"
	docker exec mms_frontend sh -c 'cd /app && pnpm install --frozen-lockfile && npx nuxi build'

	echo "== 前端：推送产物"
	rsync -az --delete --exclude '.DS_Store' --exclude '._*' \
		frontend/.output/ "${SERVER}:${APP_DIR}/frontend/.output/"

	echo "== 前端：重启"
	ssh "${SERVER}" "docker restart mms_frontend"
fi

if [ "${PART}" != "frontend" ]; then
	echo "== 后端：本机容器内交叉编译"
	# 服务器为 linux/amd64；纯 Go 依赖，关 CGO 直接交叉编译
	docker exec mms_backend sh -c 'cd /app && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server'

	echo "== 后端：服务器留回滚快照（二进制 + 数据库）"
	ssh "${SERVER}" "
		set -e
		mkdir -p '${BACKUP_DIR}'
		[ ! -f '${APP_DIR}/backend/server' ] || cp -p '${APP_DIR}/backend/server' '${BACKUP_DIR}/server_prev'
		docker exec '${PG_CONTAINER}' pg_dump -U postgres mms | gzip > '${BACKUP_DIR}/pre_deploy_\$(date +%Y%m%d_%H%M%S).sql.gz'
	"

	echo "== 后端：推送二进制并重启（goose 迁移随启动自动执行）"
	rsync -az backend/server "${SERVER}:${APP_DIR}/backend/server"
	ssh "${SERVER}" "docker restart mms_backend"
fi

echo "== 健康检查"
# 二进制秒起，但首次部署迁移建表可能较慢，最长等 90 秒
for i in $(seq 1 18); do
	sleep 5
	if curl -sf "${SITE_URL}/health" >/dev/null; then
		echo "部署完成，健康检查通过"
		exit 0
	fi
done

echo "健康检查失败（${SITE_URL}/health 不通）。回滚步骤：" >&2
echo "  1. 二进制：ssh ${SERVER} \"cp -p ${BACKUP_DIR}/server_prev ${APP_DIR}/backend/server && docker restart mms_backend\"" >&2
echo "  2. 数据库（仅当迁移损坏数据时）：用 ${BACKUP_DIR}/pre_deploy_*.sql.gz 恢复 mms 库" >&2
echo "  3. 排查：ssh ${SERVER} \"docker logs --tail 100 mms_backend\"" >&2
exit 1
