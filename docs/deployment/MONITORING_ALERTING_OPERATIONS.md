# Monitoring And Alerting Operations

This runbook defines the minimum production monitoring set for new-api before
commercial traffic is accepted.

## Required Dashboards

Track these panels per environment:

- API availability: `/api/status` success rate and latency.
- Relay availability: `/v1/chat/completions` smoke success rate by provider and model.
- Channel health: disabled channel count, auto-disable events, auto-enable events.
- Payment: created orders, successful orders, pending orders older than 10 minutes, failed callbacks.
- Billing: quota credit latency from payment callback to user quota update.
- Database: connection count, slow queries, storage usage, backup age.
- Redis: memory usage, rejected connections, command latency.
- Runtime: CPU, memory, restart count, 5xx rate.

## Alert Rules

Use these as initial thresholds, then tune after seven days of production data.

| Alert | Condition | Severity | Owner |
| --- | --- | --- | --- |
| API down | `/api/status` fails for 3 consecutive probes | P0 | on-call |
| Relay smoke failed | Provider smoke fails for 3 consecutive probes | P1 | on-call |
| High 5xx rate | 5xx rate above 1% for 5 minutes | P1 | backend |
| Payment callback failed | signed callback settlement returns `fail` | P0 | billing |
| Pending topup stale | pending online order older than 10 minutes | P1 | billing |
| Reconciliation mismatch | gateway exported amount differs from new-api CSV | P1 | billing |
| Backup stale | no successful backup in 24 hours | P0 | ops |
| Database storage high | storage above 80% | P1 | ops |
| Redis unavailable | Redis health check fails for 2 minutes | P1 | ops |

## Uptime Kuma Template

Create monitors:

- `new-api status`: HTTP GET `https://your-domain.example/api/status`, interval 60s.
- `new-api frontend`: HTTP GET `https://your-domain.example/`, interval 60s.
- `new-api provider smoke`: push monitor updated by `scripts/provider-smoke.ps1`.
- `new-api backup`: push monitor updated after `scripts/backup-postgres.ps1`.

Notification policy:

- P0: SMS or phone call plus chat notification.
- P1: chat notification with 15 minute acknowledgement target.
- P2: ticket or daily report.

## Prometheus Rule Template

```yaml
groups:
  - name: new-api-commercial-readiness
    rules:
      - alert: NewApiHighErrorRate
        expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.01
        for: 5m
        labels:
          severity: p1
        annotations:
          summary: "new-api 5xx rate is above 1%"

      - alert: NewApiBackupStale
        expr: time() - new_api_last_backup_success_timestamp > 86400
        for: 10m
        labels:
          severity: p0
        annotations:
          summary: "new-api backup is stale"

      - alert: NewApiPendingTopupStale
        expr: new_api_pending_topup_older_than_10m > 0
        for: 5m
        labels:
          severity: p1
        annotations:
          summary: "new-api has stale pending payment orders"
```

If the current stack does not export these exact metrics yet, implement the
alert with log queries or scheduled scripts first, then replace it with native
metrics after launch.

## Daily Checks

- Run provider smoke for every paid provider and production model.
- Export gateway settlement files and compare with the admin reconciliation CSV.
- Verify the latest backup file exists and the weekly restore drill has passed.
- Review disabled channels and auto-disable reasons.
