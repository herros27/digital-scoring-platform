CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competition_id UUID NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    club VARCHAR(255),
    weight_class VARCHAR(50),
    gender VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);