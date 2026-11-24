ALTER TABLE recurring_operations
    DROP CONSTRAINT IF EXISTS recurring_operations_status_check;

ALTER TABLE recurring_operations
    ADD CONSTRAINT recurring_operations_status_check
    CHECK (status IN ('active', 'paused', 'completed', 'deleted'));
