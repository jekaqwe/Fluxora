# Fluxora v1.1 deployment configuration

This file documents the external configuration required before deploying the v1.1 backend. It does not perform a deployment.

## 1. Environment file

Docker Compose reads interpolation variables from the repository-root `.env` file. Start with the provided template:

```sh
cp backend/.env.s3.example .env
```

Keep the resulting `.env` file private. It must not be committed to Git.

## 2. REG.RU S3

Create or select a **private** S3 bucket in REG.RU Cloud and configure:

- `S3_ENDPOINT=https://s3.regru.cloud`
- `S3_BUCKET`: the exact bucket name shown in the REG.RU control panel. The template uses `fluxora.su`; verify it exists.
- `S3_REGION=us-east-1`: temporary signing-region value so implementation and configuration can proceed. It is **not confirmed** as REG.RU's actual signing region. Replace it with the exact value REG.RU support confirms before relying on S3 in production.
- `S3_ACCESS_KEY_ID` and `S3_SECRET_ACCESS_KEY`: create/obtain these in the REG.RU control panel and store them only in the server-side `.env`.

The backend uses path-style addressing and AWS Signature Version 4. It stores avatars, personal/family goal images, PostgreSQL archives and account-deletion ledger records in separate object-key prefixes.

Before production, verify with test objects:
1. PUT, GET and DELETE.
2. Presigned GET links for images.
3. Object listing and seven-calendar-day backup retention.
4. Multipart upload and completion for a large test object.
5. Pending/committed deletion-ledger write, read and replay.

Never paste access keys or secret keys into ChatGPT or commit them to Git.

## 3. SMTP

Email-change confirmation uses SMTP with mandatory STARTTLS (TLS 1.2 or newer). Configure:

- `SMTP_HOST`: SMTP server hostname, without a scheme.
- `SMTP_PORT=587`: STARTTLS port.
- `SMTP_USER` and `SMTP_PASSWORD`: provider credentials, if authentication is required.
- `SMTP_FROM`: sender email address.

The backend applies connection and delivery timeouts. Test a real email-change confirmation before relying on the feature.

## 4. Backups and restore

- Daily PostgreSQL archives are scheduled for 03:00 UTC.
- If the service was offline at that time, the backend checks for today's archive on startup and creates it if missing.
- Retention keeps the current date and previous six calendar dates.
- Archives are streamed from a temporary file. Uploads above 5 GB use S3 multipart upload.
- Follow [backend/backup/RESTORE.md](backend/backup/RESTORE.md) for recovery and deletion-ledger replay.

Run a full staging restore rehearsal against the real S3 bucket before production, after replacing the temporary signing region and configuring real credentials.

## 5. Start after configuration

After the root `.env` is configured and reviewed, the deployment operator can rebuild/restart the backend using the repository's production Compose file. This repository change does not deploy or restart any running service.
