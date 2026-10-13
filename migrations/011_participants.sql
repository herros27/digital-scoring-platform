-- +goose Up
ALTER TABLE participants DROP COLUMN IF EXISTS club;
ALTER TABLE participants ALTER COLUMN name DROP NOT NULL;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS athlete_id UUID REFERENCES athletes(id) ON DELETE CASCADE;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS seed INT;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS registration_status VARCHAR(50) DEFAULT 'registered';

-- +goose Down
ALTER TABLE participants DROP COLUMN IF EXISTS registration_status;
ALTER TABLE participants DROP COLUMN IF EXISTS seed;
ALTER TABLE participants DROP COLUMN IF EXISTS athlete_id;
ALTER TABLE participants ADD COLUMN IF NOT EXISTS club VARCHAR(255);