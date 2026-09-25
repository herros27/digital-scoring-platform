CREATE TABLE athletes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    gender VARCHAR(20) NOT NULL,
    team VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);