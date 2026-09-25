CREATE TABLE match_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    round INT NOT NULL,
    athlete_red_score INT NOT NULL DEFAULT 0,
    athlete_blue_score INT NOT NULL DEFAULT 0,
    red_penalties INT NOT NULL DEFAULT 0,
    blue_penalties INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(match_id, round)
);