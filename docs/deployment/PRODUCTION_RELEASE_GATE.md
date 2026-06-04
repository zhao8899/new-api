# Production Release Gate

This gate turns the AI gateway hardening work into a repeatable release checklist. Complete it before enabling new gateway controls in production.

## 1. Configuration Gate

- [ ] `NEW_API_ENV=production` or `NEW_API_SECURITY_MODE=production` is set for production deployments.
- [ ] `SESSION_SECRET` is set to a strong random value and is not `random_string`.
- [ ] `CRYPTO_SECRET` is set to a stable strong random value.
- [ ] `SESSION_COOKIE_SECURE=true` is set when serving over HTTPS.
- [ ] `TLS_INSECURE_SKIP_VERIFY=false` is set.
- [ ] `SQL_DSN` points to PostgreSQL or MySQL for commercial production traffic.
- [ ] `REDIS_CONN_STRING` is configured for multi-instance traffic.
- [ ] `CORS_ALLOW_ORIGINS` contains explicit origins, or `CORS_ALLOW_ALL_ORIGINS=false`.
- [ ] `NEW_API_CHANNEL_CIRCUIT_MODE=observe` is used for initial rollout.
- [ ] `NEW_API_RPM_TPM_LIMIT_ENABLED=true` is only enabled with Redis available.
- [ ] `NEW_API_RPM_TPM_LIMIT_MODE=observe` is used before enforcing limits.
- [ ] `NEW_API_RPM_TPM_LIMIT_SCOPES` is narrowed deliberately for rollout.

Recommended initial gateway guardrail settings:

```env
NEW_API_CHANNEL_CIRCUIT_MODE=observe
NEW_API_RPM_TPM_LIMIT_ENABLED=true
NEW_API_RPM_TPM_LIMIT_MODE=observe
NEW_API_RPM_TPM_LIMIT_STORE_FAILURE_MODE=fail_open
NEW_API_RPM_TPM_LIMIT_SCOPES=provider,channel,model
NEW_API_RPM_LIMIT=0
NEW_API_TPM_LIMIT=0
NEW_API_RPM_TPM_LIMIT_WINDOW_SECONDS=60
```

Set non-zero `NEW_API_RPM_LIMIT` and `NEW_API_TPM_LIMIT` only after measuring normal traffic.

## 2. Security Gate

- [ ] Initial administrator creation uses setup flow and, when required, `NEW_API_SETUP_TOKEN`.
- [ ] No fixed `root / 123456` bootstrap path is relied on.
- [ ] High-risk admin routes require recent secure verification.
- [ ] WebSocket Origin allowlist behavior is validated in production mode.
- [ ] Logs redact API keys, bearer tokens, cookies, passwords, OAuth secrets, payment secrets, and provider credentials.
- [ ] Audit events redact diffs and are queryable by request id.
- [ ] Request trace error messages are redacted before storage.

Verification commands:

```bash
go test ./common ./middleware ./controller ./model
rg -n "Authorization: Bearer|sk-[A-Za-z0-9_-]{12,}|client_secret|channel_key" logs common logger controller relay service
```

Review every match from `rg`; source-code constants and tests may be acceptable, runtime logs with live secrets are not.

## 3. Gateway Stability Gate

- [ ] Provider error classification covers 401, 403, 429, 5xx, timeout, and network errors.
- [ ] Retry budget does not retry auth errors, bad requests, model-not-found errors, or content-filter errors.
- [ ] Streaming responses are not retried after bytes have been written.
- [ ] Channel health records success, failure, rate limit, timeout, auth error, server error, latency, score, and circuit state.
- [ ] Circuit breaking remains in observe mode until health events have been reviewed.
- [ ] RPM/TPM limiting remains in observe mode until thresholds have been load tested.
- [ ] `Retry-After` is returned when RPM/TPM enforcement is enabled and a request is rejected.

Verification commands:

```bash
go test ./pkg/provider ./pkg/ratelimit ./service ./model ./controller
```

## 4. Observability Gate

- [ ] A failed relay request can be found by `request_id`.
- [ ] Request trace includes model, provider, channel, latency, retry count, fallback flag, tokens, cost, and error type when available.
- [ ] Audit events can be filtered by actor, action, resource type, resource id, and request id.
- [ ] Channel health state can be inspected before switching circuit mode to `enforce`.
- [ ] Operational dashboards or log queries cover upstream success rate, first-byte latency, timeout rate, 429 rate, circuit events, and audit write failures.

Smoke checks:

```bash
curl -fsS http://localhost:3000/api/status
curl -fsS "http://localhost:3000/api/log/request_trace?p=0&page_size=10"
curl -fsS "http://localhost:3000/api/log/audit?p=0&page_size=10"
```

The log endpoints require administrator authentication in normal deployments; run the checks through the same authenticated path used by operators.

## 5. Payment Gate

- [ ] Domestic wallet top-up uses an Epay-compatible gateway with `alipay` and `wxpay` methods configured.
- [ ] Payment compliance confirmation is completed before enabling online top-up or redemption codes.
- [ ] Public callback URL is HTTPS and reachable by the payment gateway.
- [ ] `PayAddress`, `EpayId`, `EpayKey`, and callback address are configured without hardcoded secrets.
- [ ] Alipay signed callback marks one pending top-up success and credits quota exactly once.
- [ ] WeChat Pay signed callback marks one pending top-up success and credits quota exactly once.
- [ ] Duplicate payment callback is idempotent and does not credit quota twice.
- [ ] Invalid payment signature returns `fail` and does not credit quota.
- [ ] Daily reconciliation compares gateway successful trades with `topups` rows and quota deltas.
- [ ] Admin reconciliation API returns grouped totals for the target settlement window.
- [ ] Admin reconciliation CSV export is archived with the gateway settlement file.
- [ ] Missing top-up, refund, manual completion, and dispute procedures have assigned owners.

Reference: [Domestic Payment Operations](./DOMESTIC_PAYMENT_OPERATIONS.md)
Reference: [Payment Incident Runbook](./PAYMENT_INCIDENT_RUNBOOK.md)

Verification commands:

```bash
go test ./controller ./model
```

## 6. Load Test Gate

Run separate tests for:

1. Invalid token and auth failure traffic.
2. Non-stream chat completion relay.
3. Stream chat completion relay.
4. Large request bodies near configured limits.
5. Provider 429 behavior.
6. Provider 5xx or timeout behavior.
7. Admin list and search endpoints.

Track:

- p50, p95, and p99 latency
- request success rate
- 429 rate
- upstream timeout rate
- Redis pool wait
- database pool wait
- active goroutines
- process memory
- stream disconnect cleanup time

Do not enable `NEW_API_CHANNEL_CIRCUIT_MODE=enforce` or `NEW_API_RPM_TPM_LIMIT_MODE=enforce` until baseline traffic is understood.

Baseline script:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\load-test.ps1 -BaseUrl http://127.0.0.1:3000 -Path /api/status -Requests 200 -Concurrency 20
```

Provider smoke script:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\provider-smoke.ps1 -BaseUrl https://your-domain.example -ApiKey $env:NEW_API_SMOKE_KEY -Model gpt-4o-mini
```

## 7. Rollback Plan

Use configuration rollback first. Code rollback should be the last step.

| Capability | Rollback |
| --- | --- |
| Channel circuit enforcement | Set `NEW_API_CHANNEL_CIRCUIT_MODE=observe` and restart app instances. |
| RPM/TPM enforcement | Set `NEW_API_RPM_TPM_LIMIT_MODE=observe`, or `NEW_API_RPM_TPM_LIMIT_ENABLED=false`, and restart app instances. |
| Redis store failure blocking | Set `NEW_API_RPM_TPM_LIMIT_STORE_FAILURE_MODE=fail_open` and restart app instances. |
| Model/provider registry routing | Disable the registry rollout flag if configured; otherwise keep registry rows disabled and rely on legacy routing. |
| Audit write pressure | Keep query endpoints available, reduce high-volume audit middleware use, and move log storage to a separate database if needed. |
| Request trace write pressure | Move `LOG_SQL_DSN` to a separate database or reduce trace retention outside app code. |

After rollback:

- [ ] Confirm `/api/status` is healthy.
- [ ] Confirm relay success rate returns to baseline.
- [ ] Confirm no live secrets appeared in logs during the incident.
- [ ] Record the change, reason, and follow-up in the deployment changelog.

## 8. Final Verification

Run before tagging a release:

```bash
go test ./...
git diff --stat
```

Run the Docker release smoke gate from the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-smoke.ps1 -ImageTag new-api-release-smoke:current
```

The smoke gate builds the current image, starts isolated PostgreSQL/Redis/app containers, initializes the setup flow, logs in, checks authenticated APIs, registry endpoints, channel health mode, payment top-up info, payment reconciliation, and the frontend shell.

Review changed files for:

- unintended protected branding or metadata changes
- direct `encoding/json` marshal/unmarshal calls in business code
- database-specific SQL without SQLite, MySQL, and PostgreSQL compatibility
- missing tests for security, billing, rate limit, trace, or audit behavior

## 9. Backup and Recovery Gate

- [ ] SQL backups are scheduled and encrypted.
- [ ] At least one backup copy is stored outside the production host.
- [ ] A restore drill has been completed into a clean database.
- [ ] Restored user quota totals match the backup window.
- [ ] Restored top-up reconciliation totals match payment gateway records.
- [ ] Recovery steps and restore time objective are recorded.

Reference: [Backup and Recovery Operations](./BACKUP_RECOVERY_OPERATIONS.md)

PostgreSQL backup and restore drill script:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\backup-postgres.ps1 -ContainerName postgres -Database new-api -User root -RestoreDrill
```

## 10. Monitoring and Alerting Gate

- [ ] `/api/status`, frontend shell, and provider smoke monitors are configured.
- [ ] Payment callback failure and stale pending top-up alerts are configured.
- [ ] Backup stale alert is configured.
- [ ] On-call notification routes for P0 and P1 incidents are tested.
- [ ] Daily reconciliation ownership is assigned.

Reference: [Monitoring and Alerting Operations](./MONITORING_ALERTING_OPERATIONS.md)
