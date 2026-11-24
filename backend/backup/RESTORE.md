# PostgreSQL backup and restore

## Backup schedule

The backend creates a PostgreSQL custom-format archive every day at 03:00 UTC.

- S3 prefix: `backups/postgres/`
- File name: `YYYY-MM-DD.dump`
- Retention: 7 calendar days
- Expired archives are deleted after a successful upload.
- S3 credentials are read only from backend environment variables.
- Database backups are separate from user-uploaded files and avatars.

A successful backup is logged by the backend as `PostgreSQL archive uploaded`. A missing S3 configuration or failed upload is logged and does not stop the API.

## Restore procedure

1. Stop the backend so no application writes occur during restoration.
2. Download the required `backups/postgres/YYYY-MM-DD.dump` archive from the private REG.RU S3 bucket to the server.
3. Restore it into the PostgreSQL database. Run this from the directory containing the production Compose file:

   ```sh
   cat /path/to/YYYY-MM-DD.dump | docker compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_restore --clean --if-exists --no-owner --no-privileges -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
   ```

4. Reapply account deletions that happened after the selected backup. The deletion ledger is stored separately in the private S3 prefix `deletion-ledger/`, outside PostgreSQL backups:

   ```sh
   docker compose run --rm -e APPLY_DELETION_LEDGER=1 backend
   ```

   This one-off command runs database migrations, checks the external deletion ledger, and removes any account from the restored database whose user ID and email hash match a committed deletion entry. It does not start the API.

5. Start the backend again:

   ```sh
   docker compose up -d backend
   ```

6. Verify login, account balances, operations, family membership, goals, and file access before reopening the service.

## Important notes

- The PostgreSQL archive does not contain S3 user files. Those remain in the S3 bucket and are managed separately.
- Account deletion removes personal data from the active database immediately. Old PostgreSQL archives can retain the deleted data until their seven-day retention expires.
- The external deletion ledger is intentionally separate from database backups so restoring an older archive does not silently resurrect a deleted account.
- A restore test should be performed against a separate staging database before using this procedure on production.
