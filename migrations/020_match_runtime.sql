-- +goose Up
-- Keadaan pertandingan yang sedang berlangsung (state mesin, sisa waktu, sanksi, hitungan teknik,
-- keputusan wasit, dan ruleset yang dipakai saat dimuat) disimpan agar server yang restart dapat
-- memulihkan partai. Satu baris per partai aktif; dihapus saat partai selesai, dibatalkan, atau
-- dijadwalkan ulang.
CREATE TABLE IF NOT EXISTS match_runtime (
    match_id UUID PRIMARY KEY REFERENCES matches(id) ON DELETE CASCADE,
    arena_id TEXT NOT NULL,
    state VARCHAR(20) NOT NULL,
    timer_remaining_ms BIGINT NOT NULL DEFAULT 0,
    timer_running BOOLEAN NOT NULL DEFAULT FALSE,
    data JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS match_runtime;
