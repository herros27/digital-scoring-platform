ALTER TABLE competitions ADD COLUMN ruleset_id UUID REFERENCES rulesets(id);
-- optionally keep ruleset_name for backward compat during migration