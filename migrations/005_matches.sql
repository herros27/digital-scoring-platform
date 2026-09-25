CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competition_id UUID NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    arena_id UUID REFERENCES arenas(id),
    bracket_slot_id UUID REFERENCES bracket_slots(id),
    athlete_red_id UUID REFERENCES participants(id),
    athlete_blue_id UUID REFERENCES participants(id),
    current_round INT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'idle',
    winner_id UUID REFERENCES participants(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);