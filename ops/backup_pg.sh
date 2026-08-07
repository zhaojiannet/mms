#!/usr/bin/env bash
# PG mms 库定时备份脚本
#
# 部署：crontab -e 加一行（PG 容器名与默认不同时用环境变量传）
#   0 3 * * * PG_CONTAINER=1Panel-postgresql-xxxx /path/to/mms/ops/backup_pg.sh >> /var/log/mms_backup.log 2>&1
#
# 保留 14 天本地备份；可加 rclone 推 S3/阿里 OSS

# pipefail：pg_dump 失败而 gzip 成功时管道整体不能算成功，否则产出空备份还报正常
set -eo pipefail

PG_CONTAINER=${PG_CONTAINER:-postgres-server}
PG_SUPERUSER=${PG_SUPERUSER:-postgres}
DB_NAME=${DB_NAME:-mms}
BACKUP_DIR=${BACKUP_DIR:-/backups/mms}
KEEP_DAYS=${KEEP_DAYS:-14}
TS=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

OUT="$BACKUP_DIR/${DB_NAME}_${TS}.sql.gz"

echo "[$(date +'%F %T')] backup → $OUT"
docker exec "$PG_CONTAINER" pg_dump \
  -U "$PG_SUPERUSER" \
  --no-owner \
  --no-privileges \
  --format=plain \
  "$DB_NAME" \
  | gzip > "$OUT"

SIZE=$(du -h "$OUT" | cut -f1)
echo "[$(date +'%F %T')] done, size=$SIZE"

# 清理超期备份
find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" -mtime +$KEEP_DAYS -print -delete

# （可选）推送到对象存储：
# rclone copy "$OUT" oss:mms-backups/$(date +%Y/%m)/
