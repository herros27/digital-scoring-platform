-- +goose Up
-- Kategori resmi PB IPSI / PERSILAT 2026: golongan umur, rentang bobot, mode bagan.
ALTER TABLE categories ADD COLUMN IF NOT EXISTS age_group VARCHAR(50) NOT NULL DEFAULT 'dewasa';
ALTER TABLE categories ADD COLUMN IF NOT EXISTS preset_standard VARCHAR(50) NOT NULL DEFAULT 'ipsi';
ALTER TABLE categories ADD COLUMN IF NOT EXISTS min_weight NUMERIC(5,2);
ALTER TABLE categories ADD COLUMN IF NOT EXISTS max_weight NUMERIC(5,2);
ALTER TABLE categories ADD COLUMN IF NOT EXISTS bracket_mode VARCHAR(30) NOT NULL DEFAULT 'single_elimination';

-- Identitas atlet. Nullable agar data lama tetap valid; wajib diisi lewat API untuk atlet baru.
ALTER TABLE athletes ADD COLUMN IF NOT EXISTS birth_date DATE;
ALTER TABLE athletes ADD COLUMN IF NOT EXISTS nik VARCHAR(50);

-- Peserta terhubung ke kategori spesifik + hasil timbang + checklist berkas.
ALTER TABLE participants ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS actual_weight NUMERIC(5,2);
ALTER TABLE participants ADD COLUMN IF NOT EXISTS docs_verified JSONB NOT NULL DEFAULT '{}'::jsonb;
CREATE INDEX IF NOT EXISTS idx_participants_category ON participants(category_id);

-- Bagan terisolasi per kategori.
ALTER TABLE bracket_slots ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bracket_slots_category ON bracket_slots(category_id);

-- +goose Down
DROP INDEX IF EXISTS idx_bracket_slots_category;
ALTER TABLE bracket_slots DROP COLUMN IF EXISTS category_id;
DROP INDEX IF EXISTS idx_participants_category;
ALTER TABLE participants DROP COLUMN IF EXISTS docs_verified;
ALTER TABLE participants DROP COLUMN IF EXISTS actual_weight;
ALTER TABLE participants DROP COLUMN IF EXISTS category_id;
ALTER TABLE athletes DROP COLUMN IF EXISTS nik;
ALTER TABLE athletes DROP COLUMN IF EXISTS birth_date;
ALTER TABLE categories DROP COLUMN IF EXISTS bracket_mode;
ALTER TABLE categories DROP COLUMN IF EXISTS max_weight;
ALTER TABLE categories DROP COLUMN IF EXISTS min_weight;
ALTER TABLE categories DROP COLUMN IF EXISTS preset_standard;
ALTER TABLE categories DROP COLUMN IF EXISTS age_group;
