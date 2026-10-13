-- +goose Up
ALTER TABLE competitions ADD COLUMN IF NOT EXISTS ruleset_id UUID REFERENCES rulesets(id);

-- +goose Down
ALTER TABLE competitions DROP COLUMN IF EXISTS ruleset_id;