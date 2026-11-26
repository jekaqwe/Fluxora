-- Keep transaction history intact when an account is deleted.
-- Deferred NO ACTION still permits deleting a user: the user's transactions
-- and accounts are both removed by their respective user_id cascades in the
-- same transaction, while deleting an account alone is rejected at commit.
ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_account_id_fkey,
    DROP CONSTRAINT IF EXISTS transactions_transfer_account_id_fkey;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_account_id_fkey
        FOREIGN KEY (account_id) REFERENCES accounts(id)
        ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT transactions_transfer_account_id_fkey
        FOREIGN KEY (transfer_account_id) REFERENCES accounts(id)
        ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED;
