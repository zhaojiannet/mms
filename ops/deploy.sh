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
#   - 生产镜像同样本机构建后整包推送：服务器拉 Docker Hub 不可靠，且服务器上的
#     Dockerfile.prod 自 clone 后从不更新，靠它自建等于永远用旧基础镜像
#
# 安全边界：
#   - 服务器地址、域名等私域信息只在 ops/deploy.env（gitignore，不入库）
#   - rsync 永不触碰服务器上的 .env 和 uploads
set -euo pipefail
cd "$(dirname "$0")/.."

[ -f ops/deploy.env ] && . ops/deploy.env

# 服务器上 PG 容器名 / 超级用户名 / 库名（面板安装的不固定），deploy.env 可覆盖
PG_CONTAINER="${PG_CONTAINER:-postgres-server}"
PG_SUPERUSER="${PG_SUPERUSER:-postgres}"
DB_NAME="${DB_NAME:-mms}"
# 生产容器运行身份：须与服务器 compose 实际取到的 HOST_UID/HOST_GID 一致
PROD_UID="${PROD_UID:-1000}"
PROD_GID="${PROD_GID:-1000}"

for v in SERVER APP_DIR SITE_URL BACKUP_DIR; do
	if [ -z "${!v:-}" ]; then
		echo "deploy.env 缺少 ${v}。先执行：cp ops/deploy.env.example ops/deploy.env 并按注释填写" >&2
		exit 1
	fi
done

PART="${1:-all}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

# 生产镜像本机构建 → 整包推送。仅当 Dockerfile.prod 或运行身份变过才做：
# 平时一个字节都不传，改动时才付出一次约 75MB(前端)/27MB(后端) 的传输。
# 推送前把现役镜像打上时间戳标签留 3 代，新镜像出问题可退回。
sync_image() {
	local svc="$1" img="mms-${1}-prod" want have
	want=$(printf '%s|%s|%s' "$(cat "${svc}/Dockerfile.prod")" "$PROD_UID" "$PROD_GID" | shasum -a 256 | cut -d' ' -f1)
	have=$(ssh "${SERVER}" "cat '${APP_DIR}/${svc}/.image_stamp' 2>/dev/null" || true)
	if [ "$want" = "$have" ]; then
		echo "== ${svc}：生产镜像未变，跳过推送"
		return 0
	fi

	echo "== ${svc}：本机构建 linux/amd64 生产镜像"
	docker buildx build --platform linux/amd64 \
		--build-arg UID="${PROD_UID}" --build-arg GID="${PROD_GID}" \
		-f "${svc}/Dockerfile.prod" -t "${img}" --load "${svc}/"

	echo "== ${svc}：服务器留镜像回滚标签"
	# 清理旧标签的 grep 必须容错：首次部署一个 rb_ 都没有，grep 返回 1 会带着
	# set -e 把整次部署掐掉
	ssh "${SERVER}" "
		set -e
		TS=\$(date +%Y%m%d_%H%M%S)
		if docker image inspect '${img}' >/dev/null 2>&1; then
			docker tag '${img}' '${img}:rb_'\$TS
		fi
		docker images '${img}' --format '{{.Tag}}' | grep '^rb_' | sort -r | tail -n +4 \
			| xargs -r -I{} docker rmi '${img}:{}' || true
	"

	echo "== ${svc}：推送镜像"
	docker save "${img}" | gzip | ssh "${SERVER}" "gunzip | docker load"
	ssh "${SERVER}" "echo '${want}' > '${APP_DIR}/${svc}/.image_stamp'"
}

echo "== 同步部署配置（compose 与生产 Dockerfile）"
# 服务器上这几个文件靠 clone 得来、之后从不更新，不推的话镜像与编排永远停在初版
rsync -az docker-compose.yml docker-compose.prod.yml "${SERVER}:${APP_DIR}/"
rsync -az backend/Dockerfile.prod "${SERVER}:${APP_DIR}/backend/"
rsync -az frontend/Dockerfile.prod "${SERVER}:${APP_DIR}/frontend/"

if [ "${PART}" != "backend" ]; then
	echo "== 前端：本机容器内构建"
	docker exec mms_frontend sh -c 'cd /app && pnpm install --frozen-lockfile && npx nuxi build'

	echo "== 前端：推送产物"
	rsync -az --delete --exclude '.DS_Store' --exclude '._*' \
		frontend/.output/ "${SERVER}:${APP_DIR}/frontend/.output/"

	sync_image frontend

	# up -d 只负责镜像换代时重建容器；产物是 bind mount 进去的，进程不重启不会
	# 重新加载，故两条都要。--no-build：镜像缺失宁可报错，不许服务器联网自建
	echo "== 前端：重启"
	ssh "${SERVER}" "cd '${APP_DIR}' && ${COMPOSE} up -d --no-build frontend && docker restart mms_frontend"
fi

if [ "${PART}" != "frontend" ]; then
	echo "== 后端：本机容器内交叉编译"
	# 服务器为 linux/amd64；纯 Go 依赖，关 CGO 直接交叉编译
	docker exec mms_backend sh -c 'cd /app && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server'

	echo "== 后端：服务器留回滚快照（二进制 + 数据库）"
	# pipefail 必须显式开：否则 pg_dump 失败而 gzip 成功，管道整体算成功，快照悄悄丢失
	# 二进制留 3 代、库快照留 10 次：仅为快速回退，更早版本由 git 重新编译，不作存档
	# 时间戳必须在远端展开，且不能包在单引号里——单引号会让 $(date) 变成字面量，
	# 快照就都写进同一个文件名、后一次覆盖前一次（等于只剩最近一份备份）
	ssh "${SERVER}" "
		set -e -o pipefail
		BACKUP_DIR='${BACKUP_DIR}'
		APP_DIR='${APP_DIR}'
		TS=\$(date +%Y%m%d_%H%M%S)
		mkdir -p \"\$BACKUP_DIR\"
		[ ! -f \"\$APP_DIR/backend/server\" ] || cp -p \"\$APP_DIR/backend/server\" \"\$BACKUP_DIR/server_\$TS\"
		ls -t \"\$BACKUP_DIR\"/server_* 2>/dev/null | tail -n +4 | xargs -r rm --
		docker exec '${PG_CONTAINER}' pg_dump -U '${PG_SUPERUSER}' '${DB_NAME}' | gzip > \"\$BACKUP_DIR/pre_deploy_\$TS.sql.gz\"
		[ -s \"\$BACKUP_DIR/pre_deploy_\$TS.sql.gz\" ] || { echo '快照为空，中止部署' >&2; exit 1; }
		ls -t \"\$BACKUP_DIR\"/pre_deploy_*.sql.gz 2>/dev/null | tail -n +11 | xargs -r rm --
	"

	echo "== 后端：推送二进制并重启（goose 迁移随启动自动执行）"
	rsync -az backend/server "${SERVER}:${APP_DIR}/backend/server"

	sync_image backend

	ssh "${SERVER}" "cd '${APP_DIR}' && ${COMPOSE} up -d --no-build backend && docker restart mms_backend"
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
echo "  1. 二进制（最新快照即部署前在跑的那版）：" >&2
echo "     ssh ${SERVER} \"cp -p \\\$(ls -t ${BACKUP_DIR}/server_* | head -1) ${APP_DIR}/backend/server && docker restart mms_backend\"" >&2
echo "  2. 镜像（仅当本次换过 Dockerfile.prod，把 <svc> 换成 frontend 或 backend）：" >&2
echo "     ssh ${SERVER} \"cd ${APP_DIR} && docker tag mms-<svc>-prod:\\\$(docker images mms-<svc>-prod --format '{{.Tag}}' | grep '^rb_' | sort -r | head -1) mms-<svc>-prod && ${COMPOSE} up -d --no-build --force-recreate <svc>\"" >&2
echo "  3. 数据库（仅当迁移损坏数据时）：用 ${BACKUP_DIR}/pre_deploy_*.sql.gz 恢复 mms 库" >&2
echo "  4. 排查：ssh ${SERVER} \"docker logs --tail 100 mms_backend\"" >&2
echo "  提示：健康检查失败也可能出在反代/证书/DNS，与本次部署无关，先看日志再决定退不退" >&2
exit 1
