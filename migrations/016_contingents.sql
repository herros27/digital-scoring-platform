CREATE TABLE IF NOT EXISTS contingents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    pic_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE athletes ADD COLUMN IF NOT EXISTS contingent_id UUID REFERENCES contingents(id);
ALTER TABLE participants ADD COLUMN IF NOT EXISTS contingent_id UUID REFERENCES contingents(id);
