ALTER TABLE recurring_operations
    ADD COLUMN IF NOT EXISTS anchor_day SMALLINT;

ALTER TABLE recurring_operations
    ADD COLUMN IF NOT EXISTS anchor_month SMALLINT;

UPDATE recurring_operations
SET anchor_day = COALESCE(anchor_day, EXTRACT(DAY FROM next_date)::smallint),
    anchor_month = COALESCE(anchor_month, EXTRACT(MONTH FROM next_date)::smallint);

ALTER TABLE recurring_operations
    ALTER COLUMN anchor_day SET NOT NULL,
    ALTER COLUMN anchor_month SET NOT NULL;

ALTER TABLE recurring_operations
    DROP CONSTRAINT IF EXISTS recurring_anchor_day_check,
    DROP CONSTRAINT IF EXISTS recurring_anchor_month_check;

ALTER TABLE recurring_operations
    ADD CONSTRAINT recurring_anchor_day_check CHECK (anchor_day BETWEEN 1 AND 31),
    ADD CONSTRAINT recurring_anchor_month_check CHECK (anchor_month BETWEEN 1 AND 12);
