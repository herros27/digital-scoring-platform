-- +goose Up
CREATE TABLE rulesets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    scoring JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_rulesets_name_version ON rulesets(name, version);

-- +goose Down
DROP INDEX IF EXISTS idx_rulesets_name_version;
DROP TABLE IF EXISTS rulesets;