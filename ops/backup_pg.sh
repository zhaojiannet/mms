#!/usr/bin/env bash
# PG mms 库定时备份脚本
#
# 部署：crontab -e 加一行
#   0 3 * * * /path/to/mms/ops/backup_pg.sh >> /var/log/mms_backup.log 2>&1
#
# 保留 14 天本地备份；可加 rclone 推 S3/阿里 OSS

set -e

BACKUP_DIR=${BACKUP_DIR:-/backups/mms}
KEEP_DAYS=${KEEP_DAYS:-14}
TS=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

OUT="$BACKUP_DIR/mms_${TS}.sql.gz"

echo "[$(date +'%F %T')] backup → $OUT"
docker exec postgres-server pg_dump \
  -U postgres \
  --no-owner \
  --no-privileges \
  --format=plain \
  mms \
  | gzip > "$OUT"

SIZE=$(du -h "$OUT" | cut -f1)
echo "[$(date +'%F %T')] done, size=$SIZE"

# 清理超期备份
find "$BACKUP_DIR" -name 'mms_*.sql.gz' -mtime +$KEEP_DAYS -print -delete

# （可选）推送到对象存储：
# rclone copy "$OUT" oss:mms-backups/$(date +%Y/%m)/
