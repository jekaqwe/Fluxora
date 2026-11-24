-- Verify that deleting an account cannot erase transfer history, while
-- deleting the owning user still cascades cleanly across both tables.
BEGIN;

DO $$
DECLARE
    test_user_id BIGINT;
    source_account_id BIGINT;
    destination_account_id BIGINT;
    transfer_id BIGINT;
BEGIN
    INSERT INTO users(first_name, last_name, username, email, password_hash)
    VALUES ('CI', 'History', 'ci_account_history', 'ci_account_history@example.test', 'test-hash')
    RETURNING id INTO test_user_id;

    INSERT INTO accounts(user_id, name, account_type)
    VALUES (test_user_id, 'Source', 'bank')
    RETURNING id INTO source_account_id;

    INSERT INTO accounts(user_id, name, account_type)
    VALUES (test_user_id, 'Destination', 'bank')
    RETURNING id INTO destination_account_id;

    INSERT INTO transactions(
        user_id, account_id, transfer_account_id, title, operation_type,
        amount, category, operation_date, idempotency_key
    )
    VALUES (
        test_user_id, source_account_id, destination_account_id, 'Test transfer',
        'transfer', 10, 'CI', CURRENT_DATE, 'ci-account-history-transfer'
    )
    RETURNING id INTO transfer_id;

    BEGIN
        DELETE FROM accounts WHERE id = destination_account_id;
        SET CONSTRAINTS ALL IMMEDIATE;
        RAISE EXCEPTION 'deleting an account with transaction history unexpectedly succeeded';
    EXCEPTION
        WHEN foreign_key_violation THEN
            NULL;
    END;

    SET CONSTRAINTS ALL DEFERRED;

    IF NOT EXISTS (
        SELECT 1 FROM transactions
        WHERE id = transfer_id
          AND account_id = source_account_id
          AND transfer_account_id = destination_account_id
    ) THEN
        RAISE EXCEPTION 'transfer history was not preserved after rejected account deletion';
    END IF;

    -- User deletion must still remove both accounts and transactions.
    DELETE FROM users WHERE id = test_user_id;
    SET CONSTRAINTS ALL IMMEDIATE;

    IF EXISTS (SELECT 1 FROM accounts WHERE user_id = test_user_id)
       OR EXISTS (SELECT 1 FROM transactions WHERE user_id = test_user_id) THEN
        RAISE EXCEPTION 'user cascade left financial rows behind';
    END IF;
END
$$;

ROLLBACK;
