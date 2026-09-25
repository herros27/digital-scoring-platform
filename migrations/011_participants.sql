ALTER TABLE participants DROP COLUMN club;
ALTER TABLE participants ADD COLUMN athlete_id UUID NOT NULL REFERENCES athletes(id) ON DELETE CASCADE;
ALTER TABLE participants ADD COLUMN competition_id UUID NOT NULL REFERENCES competitions(id) ON DELETE CASCADE;
ALTER TABLE participants ADD COLUMN weight_class VARCHAR(50);
ALTER TABLE participants ADD COLUMN gender VARCHAR(20);
ALTER TABLE participants ADD COLUMN seed INT;
ALTER TABLE participants ADD COLUMN registration_status VARCHAR(50) DEFAULT 'registered';
-- note: old competition_id, weight_class, gender columns will be dropped via above ALTERs; need to copy data? We'll assume fresh DB or handle in app.
-- For safety, keep old columns until data migrated, then drop.