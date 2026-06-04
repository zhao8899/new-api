# Backup and Recovery Operations

Commercial deployments must treat user quota, top-up records, provider configuration, tokens, audit logs, and request traces as recoverable business data. Backups are not complete until a restore has been tested.

## Backup Scope

Back up these stores:

| Store | Required Data |
| --- | --- |
| PostgreSQL/MySQL | users, tokens, topups, subscriptions, options, channels, registries, audit/log tables |
| Redis | optional cache only; persistent business state should remain in SQL |
| `/data` | SQLite deployments, local runtime data |
| `/app/logs` | operational logs when mounted from Docker |
| `.env` / secrets | encrypted separately in the operator password manager or secret manager |

## PostgreSQL Backup

Run from the Docker host:

```bash
docker exec postgres pg_dump -U root -d new-api --format=custom --file=/tmp/new-api.dump
docker cp postgres:/tmp/new-api.dump ./backup/new-api-$(date +%Y%m%d%H%M%S).dump
```

PowerShell helper:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\backup-postgres.ps1 -ContainerName postgres -Database new-api -User root
```

Restore drill into a clean database:

```bash
docker exec postgres createdb -U root new-api-restore
docker cp ./backup/new-api.dump postgres:/tmp/new-api.dump
docker exec postgres pg_restore -U root -d new-api-restore --clean --if-exists /tmp/new-api.dump
```

PowerShell helper with restore drill:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\backup-postgres.ps1 -ContainerName postgres -Database new-api -User root -RestoreDrill
```

Validate after restore:

```sql
select count(*) from users;
select count(*) from top_ups;
select count(*) from options;
select sum(quota) from users;
```

## MySQL Backup

```bash
docker exec mysql mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" new-api > ./backup/new-api-$(date +%Y%m%d%H%M%S).sql
```

Restore drill:

```bash
docker exec mysql mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "create database if not exists new_api_restore"
docker exec -i mysql mysql -uroot -p"$MYSQL_ROOT_PASSWORD" new_api_restore < ./backup/new-api.sql
```

## Retention

- Keep hourly backups for 24 hours.
- Keep daily backups for 30 days.
- Keep monthly backups for 12 months.
- Encrypt backups before moving them off host.
- Store at least one copy outside the production host.

## Recovery Checklist

- [ ] Restore SQL backup into a clean database.
- [ ] Start a temporary app instance against the restored database.
- [ ] Confirm `/api/status` returns success.
- [ ] Confirm root/admin login works.
- [ ] Confirm user quota totals match the source backup window.
- [ ] Confirm top-up reconciliation totals match payment gateway records.
- [ ] Confirm provider/channel/model registry settings are present.
- [ ] Record restore time objective and any manual steps.

## Failure Handling

If a restore fails:

1. Stop rollout or incident recovery until the failure is understood.
2. Preserve the failed backup file and restore logs.
3. Try the previous backup in the retention chain.
4. Compare `top_ups`, `users.quota`, and subscription rows before reopening paid traffic.
