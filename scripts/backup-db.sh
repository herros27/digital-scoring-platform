#!/usr/bin/env bash
# ponytail: Skrip backup PostgreSQL digital_scoring ke file terkompresi gzip
# Bisa dipasang di crontab: 0 2 * * * /path/to/DigitalScoring/scripts/backup-db.sh
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-$(dirname "$0")/../backups}"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/digital_scoring_${TIMESTAMP}.sql.gz"

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Membuat cadangan database ke $BACKUP_FILE..."
docker exec scoring-postgres pg_dump -U scoring_user -d digital_scoring | gzip > "$BACKUP_FILE"

# Simpan cadangan 7 hari terakhir, hapus yang lebih lama
find "$BACKUP_DIR" -type f -name "digital_scoring_*.sql.gz" -mtime +7 -delete 2>/dev/null || true

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Cadangan berhasil dibuat ($BACKUP_FILE)"

