ALTER TABLE family_transactions
    DROP CONSTRAINT IF EXISTS family_transactions_user_idempotency_unique;

ALTER TABLE family_transactions
    ADD CONSTRAINT family_transactions_family_author_idempotency_unique
    UNIQUE (family_id, author_user_id, idempotency_key);
