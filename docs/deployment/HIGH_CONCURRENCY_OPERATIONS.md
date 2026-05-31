# High-Concurrency Operations Guide

This guide defines the baseline deployment and tuning requirements for running new-api under sustained high-concurrency API traffic.

## Target Architecture

Use a layered deployment:

- Load balancer or reverse proxy: terminates TLS, applies connection limits, and forwards to app instances.
- App instances: stateless new-api containers or processes.
- Redis: shared cache, distributed rate limiting, and cross-instance coordination.
- PostgreSQL or MySQL: primary durable database.
- Optional separate log database: isolates audit/log write load from primary business data.

Do not use SQLite for commercial high-concurrency deployments. SQLite is suitable for local, demo, and small single-node use only.

## Required Runtime Settings

Set these in production:

```env
NEW_API_ENV=production
GIN_MODE=release
SESSION_SECRET=<strong-random-secret>
CRYPTO_SECRET=<stable-strong-random-secret>
SESSION_COOKIE_SECURE=true
TLS_INSECURE_SKIP_VERIFY=false
```

For API gateway traffic, start with:

```env
RELAY_TIMEOUT=120
STREAMING_TIMEOUT=300
MAX_REQUEST_BODY_MB=16

HTTP_READ_HEADER_TIMEOUT=10
HTTP_READ_TIMEOUT=0
HTTP_WRITE_TIMEOUT=0
HTTP_IDLE_TIMEOUT=120
HTTP_SHUTDOWN_TIMEOUT=30

RELAY_MAX_IDLE_CONNS=2000
RELAY_MAX_IDLE_CONNS_PER_HOST=200
```

Keep `HTTP_WRITE_TIMEOUT=0` when serving long SSE streams directly from the app. If a reverse proxy owns stream timeouts, configure both layers deliberately and test client disconnect behavior.

## Redis Sizing

Redis must be enabled for multi-instance and high-concurrency deployments.

```env
REDIS_CONN_STRING=redis://:<password>@redis:6379/0
REDIS_POOL_SIZE=100
REDIS_MIN_IDLE_CONNS=10
REDIS_POOL_TIMEOUT=5
```

Tune upward when Redis pool wait time appears in traces or request latency. A practical starting point is:

- small deployment: `REDIS_POOL_SIZE=50`
- moderate deployment: `REDIS_POOL_SIZE=100-200`
- large deployment: size by measured concurrent Redis operations, not request count alone

Do not fall back to in-memory rate limiting for commercial multi-instance traffic. The in-memory limiter is process-local and has a global mutex.

## Database Sizing

Use PostgreSQL or MySQL with explicit connection limits:

```env
SQL_MAX_OPEN_CONNS=200
SQL_MAX_IDLE_CONNS=50
SQL_MAX_LIFETIME=300
```

Avoid setting `SQL_MAX_OPEN_CONNS` higher than the database can actually serve. For multiple app instances:

```text
total_possible_connections = app_instance_count * SQL_MAX_OPEN_CONNS
```

Keep that number below the database server limit with headroom for migrations, admin sessions, and background jobs.

## Reverse Proxy Requirements

The reverse proxy should enforce:

- TLS termination with modern protocols.
- Request body size consistent with `MAX_REQUEST_BODY_MB`.
- Header read timeout to reduce slowloris exposure.
- Long-lived stream support for SSE and WebSocket routes.
- Connection and request rate limits per IP or tenant where possible.
- Health checks against `/api/status`.

Recommended behavior:

```text
client_body_timeout: 30s
client_header_timeout: 10s
keepalive_timeout: 120s
proxy_read_timeout for stream routes: >= STREAMING_TIMEOUT
max body size: same or lower than MAX_REQUEST_BODY_MB
```

## App Instance Scaling

Scale app instances horizontally when any of these holds for a sustained period:

- CPU > 70%
- memory pressure or frequent GC pauses
- upstream relay latency increases while Redis and DB are healthy
- goroutine count grows with active stream count and does not return after disconnects
- DB or Redis pools are saturated

Long-lived stream requests consume more resources than short JSON requests. Capacity planning must measure stream and non-stream traffic separately.

## Load Test Plan

Run separate tests for each traffic class:

1. Auth failures and invalid token traffic.
2. Non-stream chat completion relay.
3. Stream chat completion relay.
4. Large request bodies near configured limits.
5. Model list and token usage endpoints.
6. Payment webhook bursts.
7. Admin search/list endpoints.

Track:

- p50/p95/p99 latency
- error rate
- active goroutines
- open file descriptors
- process RSS
- DB pool in-use/wait stats
- Redis pool hits/misses/timeouts
- upstream provider error rate
- stream disconnect cleanup time

## Operating Limits

Define explicit rollout limits before launch:

```text
max app instances:
max concurrent non-stream requests per instance:
max concurrent stream requests per instance:
max request body size:
max Redis pool wait:
max DB connection pool wait:
max acceptable p99 latency:
```

Do not claim a global concurrency number without a reproducible load test report. The safe capacity depends on model mix, stream ratio, upstream latency, DB size, Redis size, and reverse proxy settings.

## Production Readiness Checklist

- Redis enabled and sized.
- PostgreSQL or MySQL configured; SQLite disabled.
- App server timeouts configured.
- Reverse proxy stream and body limits aligned with app settings.
- `SESSION_SECRET` and `CRYPTO_SECRET` explicitly set.
- `SESSION_COOKIE_SECURE=true` under HTTPS.
- `TLS_INSECURE_SKIP_VERIFY=false`.
- `RELAY_TIMEOUT` set for non-local deployments.
- `/api/status` health check configured.
- Load test covers both stream and non-stream traffic.
- Known failing tests are documented before release.

