<#
.SYNOPSIS
  Menyalakan ulang backend Digital Scoring (Windows): mematikan yang sedang berjalan, membangun ulang, menyalakan
  lagi di latar belakang, dan BARU selesai setelah server siap. Dipakai spec E2E pemulihan restart.

.DESCRIPTION
  Dari folder apps/web (PowerShell):
    $env:E2E_RESTART_CMD = 'powershell -NoProfile -ExecutionPolicy Bypass -File ..\..\scripts\restart-backend.ps1'
    npm run test:match-flow -- 06

  Backend dijalankan dari akar repo sehingga `.env` terbaca seperti `go run ./cmd/server`. Database harus sudah hidup.
  Yang dimatikan: proses dari skrip ini (bin\backend.pid) dan proses apa pun yang mendengarkan port backend
  (mis. `go run` yang dijalankan manual). Log: bin\backend.log (stdout) dan bin\backend.err.log (stderr).
  Kode keluar 0 = server siap, 1 = gagal (log terakhir dicetak).
#>
param(
  [int]$Port = 0,
  [int]$TimeoutSec = 60,
  [switch]$NoBuild
)

$ErrorActionPreference = 'Stop'

$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$BinDir = Join-Path $Root 'bin'
$Exe = Join-Path $BinDir 'server.exe'
$PidFile = Join-Path $BinDir 'backend.pid'
$OutLog = Join-Path $BinDir 'backend.log'
$ErrLog = Join-Path $BinDir 'backend.err.log'

function Write-Step([string]$msg) { Write-Host "[restart-backend] $msg" }

if ($Port -le 0 -and $env:PORT) { $Port = [int]$env:PORT }
if ($Port -le 0) { # PORT dari .env bila tidak diberikan
  $envFile = Join-Path $Root '.env'
  if (Test-Path $envFile) {
    $line = Get-Content $envFile | Where-Object { $_ -match '^\s*PORT\s*=' } | Select-Object -Last 1
    if ($line) {
      $val = ($line -replace '^\s*PORT\s*=', '').Trim().Trim('"').Trim("'")
      if ($val -match '^\d+$') { $Port = [int]$val }
    }
  }
}
if ($Port -le 0) { $Port = 8080 }
$Health = "http://127.0.0.1:$Port/api/v1/health"

function Get-ListenerPids {
  $found = @()
  try {
    $found = @(Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop | Select-Object -ExpandProperty OwningProcess)
  } catch {
    # tanpa modul NetTCPConnection: baca keluaran netstat
    $found = @(netstat -ano -p tcp | ForEach-Object {
        if ($_ -match "^\s*TCP\s+\S+:$Port\s+\S+\s+LISTENING\s+(\d+)\s*$") { [int]$Matches[1] }
      })
  }
  return @($found | Where-Object { $_ -and $_ -gt 0 } | Sort-Object -Unique)
}

function Stop-Backend {
  if (Test-Path $PidFile) {
    $old = (Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
    if ($old -match '^\d+$') { Stop-Process -Id ([int]$old) -Force -ErrorAction SilentlyContinue }
    Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
  }
  foreach ($p in Get-ListenerPids) { Stop-Process -Id $p -Force -ErrorAction SilentlyContinue }
  for ($i = 0; $i -lt 50; $i++) { # tunggu port bebas (maks 5 dtk)
    if ((Get-ListenerPids).Count -eq 0) { return }
    Start-Sleep -Milliseconds 100
  }
}

function Show-LogTail {
  foreach ($f in @($OutLog, $ErrLog)) {
    if ((Test-Path $f) -and (Get-Item $f).Length -gt 0) {
      Write-Host "--- $(Split-Path $f -Leaf) ---"
      Get-Content $f -Tail 25 | ForEach-Object { Write-Host $_ }
    }
  }
}

Write-Step "mematikan backend di port $Port ..."
Stop-Backend

if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }

if (-not $NoBuild) {
  Write-Step 'membangun ...'
  Push-Location $Root
  try {
    & go build -o $Exe ./cmd/server
    if ($LASTEXITCODE -ne 0) { Write-Host '[restart-backend] go build gagal' -ForegroundColor Red; exit 1 }
  } finally { Pop-Location }
}
if (-not (Test-Path $Exe)) {
  Write-Host "[restart-backend] $Exe tidak ada (jalankan tanpa -NoBuild)" -ForegroundColor Red
  exit 1
}

Write-Step 'menyalakan ...'
# Jendela tersembunyi + stdout/stderr ke berkas: tetap hidup setelah skrip selesai dan tidak menahan pipa pemanggil
$proc = Start-Process -FilePath $Exe -WorkingDirectory $Root -WindowStyle Hidden -PassThru `
  -RedirectStandardOutput $OutLog -RedirectStandardError $ErrLog
Set-Content -Path $PidFile -Value $proc.Id

$deadline = (Get-Date).AddSeconds($TimeoutSec)
while ((Get-Date) -lt $deadline) {
  if ($proc.HasExited) {
    Write-Host "[restart-backend] backend berhenti saat start (kode $($proc.ExitCode)). Log terakhir:" -ForegroundColor Red
    Show-LogTail
    exit 1
  }
  try {
    $r = Invoke-WebRequest -Uri $Health -UseBasicParsing -TimeoutSec 2
    if ($r.StatusCode -eq 200) {
      Write-Step "siap di http://127.0.0.1:$Port (PID $($proc.Id), log: $OutLog)"
      exit 0
    }
  } catch { } # belum siap
  Start-Sleep -Milliseconds 250
}
Write-Host "[restart-backend] tidak siap dalam $TimeoutSec dtk. Log terakhir:" -ForegroundColor Red
Show-LogTail
exit 1
