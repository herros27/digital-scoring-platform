-- +goose Up
-- Kunci draw bagan per kategori, disimpan di server. Sebelumnya "Kunci Draw" hanya state di browser panitia
-- sehingga tidak tersimpan dan tidak ditegakkan: bagan resmi tetap bisa diacak ulang lewat API. NULL = terbuka.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS bracket_locked_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE categories DROP COLUMN IF EXISTS bracket_locked_at;
