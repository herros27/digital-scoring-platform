-- +goose Up
-- Gelanggang asal kategori: semua partai satu kategori (termasuk semifinal dan final) dimainkan di gelanggang ini.
-- Diisi otomatis saat bagan dibuat (beban paling ringan) dan dapat diubah panitia sebelum partai berjalan.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS home_arena_id UUID REFERENCES arenas(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE categories DROP COLUMN IF EXISTS home_arena_id;
