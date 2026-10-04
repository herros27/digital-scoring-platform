-- +goose Up
ALTER TABLE judge_pins ADD COLUMN IF NOT EXISTS seat_number INT DEFAULT 0;

-- +goose Down
ALTER TABLE judge_pins DROP COLUMN IF EXISTS seat_number;
