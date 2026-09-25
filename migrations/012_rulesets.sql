CREATE TABLE rulesets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    scoring JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- unique constraint on name+version
CREATE UNIQUE INDEX idx_rulesets_name_version ON rulesets(name, version);