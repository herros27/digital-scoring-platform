#!/usr/bin/env bash
# Menyalakan ulang backend Digital Scoring (Linux/macOS): mematikan yang sedang berjalan, membangun ulang, menyalakan
# lagi di latar belakang, dan BARU selesai setelah server siap. Dipakai spec E2E pemulihan restart:
#
#   E2E_RESTART_CMD="$PWD/scripts/restart-backend.sh" npm run test:match-flow -- 06      # di apps/web
#
# Backend dijalankan dari akar repo sehingga `.env` terbaca seperti `go run ./cmd/server`. Database harus sudah hidup.
# Yang dimatikan: proses dari skrip ini (bin/backend.pid) dan proses apa pun yang mendengarkan port backend
# (mis. `go run` yang dijalankan manual). Opsi lewat variabel: PORT (bawaan 8080), TIMEOUT_SEC (bawaan 60), NO_BUILD=1,
# CONTAINER=<nama> (backend di Docker: restart kontainer itu saja, tanpa build/kill proses).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT/bin"
EXE="$BIN_DIR/server"
PID_FILE="$BIN_DIR/backend.pid"
LOG="$BIN_DIR/backend.log"
PORT="${PORT:-}"
if [ -z "$PORT" ] && [ -f "$ROOT/.env" ]; then # PORT dari .env bila tidak diberikan lewat lingkungan
  PORT="$(grep -E '^PORT=' "$ROOT/.env" | tail -1 | cut -d= -f2 | tr -d '[:space:]"' || true)"
fi
PORT="${PORT:-8080}"
TIMEOUT_SEC="${TIMEOUT_SEC:-60}"
HEALTH="http://127.0.0.1:${PORT}/api/v1/health"

listener_pids() { # PID yang mendengarkan TCP $PORT
  if command -v lsof >/dev/null 2>&1; then
    lsof -t -iTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true
  elif command -v ss >/dev/null 2>&1; then
    ss -ltnpH "sport = :$PORT" 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u || true
  elif command -v fuser >/dev/null 2>&1; then
    fuser -n tcp "$PORT" 2>/dev/null | tr -s ' ' '\n' | grep -E '^[0-9]+$' || true
  fi
}

stop_backend() {
  if [ -f "$PID_FILE" ]; then
    old="$(cat "$PID_FILE" 2>/dev/null || true)"
    [ -n "$old" ] && kill "$old" 2>/dev/null || true
    rm -f "$PID_FILE"
  fi
  # Hanya matikan proses backend (server / go run). Pendengar lain, mis. docker-proxy bila backend ada di kontainer,
  # tidak boleh dimatikan: pakai CONTAINER=<nama> untuk backend di Docker.
  for p in $(listener_pids); do
    name="$(ps -o comm= -p "$p" 2>/dev/null | tr -d ' ')"
    case "$name" in
      server|main|go|"") kill "$p" 2>/dev/null || true ;;
      *) echo "[restart-backend] port $PORT dipakai proses lain: $name (PID $p). Tidak dimatikan. Backend di Docker? Pakai CONTAINER=<nama>." >&2; exit 1 ;;
    esac
  done
  for _ in $(seq 1 50); do # tunggu port bebas (maks 5 dtk), lalu paksa
    [ -z "$(listener_pids)" ] && return 0
    sleep 0.1
  done
  for p in $(listener_pids); do kill -9 "$p" 2>/dev/null || true; done # tersisa hanya proses yang lolos pemeriksaan di atas
  sleep 0.5
}

if [ -n "${CONTAINER:-}" ]; then # backend di Docker: restart kontainer lalu tunggu health
  echo "[restart-backend] restart kontainer $CONTAINER ..."
  docker restart "$CONTAINER" >/dev/null || { echo "[restart-backend] docker restart gagal" >&2; exit 1; }
  for _ in $(seq 1 $((TIMEOUT_SEC * 4))); do
    if curl -fsS -o /dev/null --max-time 2 "$HEALTH" 2>/dev/null; then
      echo "[restart-backend] siap di http://127.0.0.1:${PORT} (kontainer $CONTAINER)"
      exit 0
    fi
    sleep 0.25
  done
  echo "[restart-backend] tidak siap dalam ${TIMEOUT_SEC} dtk. Log kontainer:" >&2
  docker logs --tail 25 "$CONTAINER" >&2 || true
  exit 1
fi

echo "[restart-backend] mematikan backend di port $PORT ..."
stop_backend

mkdir -p "$BIN_DIR"
if [ "${NO_BUILD:-}" != "1" ]; then
  echo "[restart-backend] membangun ..."
  (cd "$ROOT" && go build -o "$EXE" ./cmd/server)
fi
[ -x "$EXE" ] || { echo "[restart-backend] $EXE tidak ada (jalankan tanpa NO_BUILD=1)" >&2; exit 1; }

echo "[restart-backend] menyalakan ..."
# setsid + nohup + stdin/stdout/stderr dilepas: tetap hidup setelah skrip selesai dan tidak menahan pipa pemanggil
# exec supaya PID yang dicatat adalah PID server itu sendiri, bukan subshell yang menahan pipa pemanggil
(cd "$ROOT" && exec setsid nohup "$EXE" >"$LOG" 2>&1 </dev/null) &
PID=$!
echo "$PID" >"$PID_FILE"
disown "$PID" 2>/dev/null || true

for _ in $(seq 1 $((TIMEOUT_SEC * 4))); do
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "[restart-backend] backend berhenti saat start. Log terakhir:" >&2
    tail -n 25 "$LOG" >&2 || true
    exit 1
  fi
  if curl -fsS -o /dev/null --max-time 2 "$HEALTH" 2>/dev/null; then
    echo "[restart-backend] siap di http://127.0.0.1:${PORT} (PID $PID, log: $LOG)"
    exit 0
  fi
  sleep 0.25
done
echo "[restart-backend] tidak siap dalam ${TIMEOUT_SEC} dtk. Log terakhir:" >&2
tail -n 25 "$LOG" >&2 || true
exit 1
