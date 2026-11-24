# PostgreSQL backup and restore

## Backup schedule

The backend creates a PostgreSQL custom-format archive every day at 03:00 UTC.

- S3 prefix: `backups/postgres/`
- File name: `YYYY-MM-DD.dump`
- Retention: seven calendar dates, including the current backup date
- Expired archives are deleted after a successful upload and successful listing.
- S3 credentials are read only from backend environment variables.
- Database backups are separate from user-uploaded files and avatars.

The backend writes `pg_dump` to a temporary file and uploads it to S3 from that file. It does not hold the entire database archive in process memory. A successful backup is logged as `PostgreSQL archive uploaded`. A missing S3 configuration or failed upload is logged and does not stop the API.

## Restore procedure

Perform a restore rehearsal against a separate staging database before any production recovery.

1. Stop the backend so no application writes occur during restoration.
2. Download the required private S3 object `backups/postgres/YYYY-MM-DD.dump` to the server. Do not place it in a public web directory.
3. Restore it into the intended PostgreSQL database. Run from the directory containing the relevant Compose file, after checking that its service and database names match this environment:

   ```sh
   docker compose stop backend
   docker compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_restore --clean --if-exists --no-owner --no-privileges -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < /path/to/YYYY-MM-DD.dump
   ```

4. Reapply committed account deletions recorded after the selected backup. The deletion ledger is stored separately in private S3 under `deletion-ledger/committed/`. The restore check matches the immutable user ID and the account's original `users.created_at` timestamp. It does not use email, because an email address can change.

   Run the one-off backend process with the same database and S3 environment as the application:

   ```sh
   docker compose run --rm -e APPLY_DELETION_LEDGER=1 backend
   ```

   The process applies migrations, reads only the committed ledger prefix, reapplies each deletion for the matching account incarnation, then exits without starting the HTTP API. Pending ledger entries are not treated as completed deletions.

5. Start the backend again:

   ```sh
   docker compose up -d backend
   ```

6. Verify login, account balances, operations, family membership and ownership, goals, recurring operations, and file access before reopening the service.

## Important notes

- The PostgreSQL archive does not contain S3 user files. Those remain in the S3 bucket and are managed separately.
- Account deletion removes personal data from the active database immediately. Old PostgreSQL archives can retain deleted data until their seven-day retention expires.
- The external deletion ledger is intentionally separate from database backups so restoring an older archive does not silently resurrect a deleted account.
- If account deletion reports a ledger warning, do not restore a database backup until the ledger issue has been investigated and corrected.
- The one-off ledger replay needs working REG.RU S3 credentials, the correct bucket and signing region, and access to the committed deletion prefix.
- A restore rehearsal should include an account deleted after the chosen backup, and should confirm that the account stays deleted after ledger replay.
