ALTER TABLE family_goal_entries
    ADD COLUMN IF NOT EXISTS entry_type VARCHAR(12) NOT NULL DEFAULT 'contribution';

ALTER TABLE family_goal_entries
    DROP CONSTRAINT IF EXISTS family_goal_entries_entry_type_check;

ALTER TABLE family_goal_entries
    ADD CONSTRAINT family_goal_entries_entry_type_check
    CHECK (entry_type IN ('contribution', 'withdrawal'));

