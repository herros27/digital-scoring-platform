CREATE TABLE bracket_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competition_id UUID NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    weight_class VARCHAR(50) NOT NULL,
    gender VARCHAR(20) NOT NULL,
    round INT NOT NULL,
    slot_index INT NOT NULL,
    participant_id UUID REFERENCES participants(id),
    is_bye BOOLEAN NOT NULL DEFAULT FALSE,
    winner_id UUID REFERENCES participants(id),
    next_slot_id UUID REFERENCES bracket_slots(id)
);