-- +goose Up
ALTER TABLE judge_pins ADD COLUMN IF NOT EXISTS raw_pin VARCHAR(20);

-- +goose Down
ALTER TABLE judge_pins DROP COLUMN IF EXISTS raw_pin;
