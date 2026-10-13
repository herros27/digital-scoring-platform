param(
    [string]$BackupDir = "$PSScriptRoot/../backups"
)

# ponytail: Skrip backup PostgreSQL digital_scoring ke file terkompresi / sql
if (!(Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$backupFile = Join-Path $BackupDir "digital_scoring_$timestamp.sql"

Write-Host "[(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Membuat cadangan database ke $backupFile..."
docker exec scoring-postgres pg_dump -U scoring_user -d digital_scoring > $backupFile

# Bersihkan backup lebih lama dari 7 hari
Get-ChildItem -Path $BackupDir -Filter "digital_scoring_*.sql" |
    Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-7) } |
    Remove-Item -Force -ErrorAction SilentlyContinue

Write-Host "[(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] Cadangan berhasil dibuat: $backupFile"

