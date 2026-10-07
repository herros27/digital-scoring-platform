-- +goose Up
-- Login juri/wasit kini memakai PIN saja: server mencari PIN lewat pin_lookup
-- (HMAC-SHA256 dari PIN, lihat auth.PinLookup) yang harus unik.
--
-- Nilai pin_lookup tidak bisa dihitung dari SQL (butuh secret server), sehingga PIN yang sudah ada
-- dibuang. Sistem masih tahap development; PIN dibuat ulang otomatis lewat GET /auth/pins atau seeder.
DELETE FROM judge_pins;
ALTER TABLE judge_pins ADD COLUMN IF NOT EXISTS pin_lookup CHAR(64) NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_judge_pins_pin_lookup ON judge_pins (pin_lookup);

-- +goose Down
DROP INDEX IF EXISTS ux_judge_pins_pin_lookup;
ALTER TABLE judge_pins DROP COLUMN IF EXISTS pin_lookup;
