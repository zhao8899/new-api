# Windows Deployment Checklist

## Purpose

This checklist is the execution companion to:

- [SEATTLE_RESIDENTIAL_DEPLOYMENT_MANUAL.md](./SEATTLE_RESIDENTIAL_DEPLOYMENT_MANUAL.md)
- [HEADER_FORWARDING_OPTIMIZATION_RECORD.md](../security/HEADER_FORWARDING_OPTIMIZATION_RECORD.md)

It consolidates:

- Windows host preparation
- Docker Desktop / WSL2 runtime checks
- `new-api` deployment checks
- Cloudflare Tunnel checks
- Cloudflare Access checks
- validation and recovery checks

This checklist supersedes the old dashboard-only checklist so there is only one operational checklist to maintain.

## Environment Record

Record these before execution:

- host name: `__________`
- Windows account: `__________`
- Cloudflare account: `__________`
- Zero Trust org: `__________`
- root domain: `__________`
- admin hostname: `__________`
- api hostname: `__________`
- deployment path: `__________`
- local service port: `__________`
- backup maintenance path: `__________`

Do not store real secrets in the repository.

## 1. Host Prerequisites

### 1.1 Power and Availability

- [ ] laptop stays plugged in
- [ ] sleep is disabled
- [ ] hibernation is disabled
- [ ] lid-close sleep is disabled
- [ ] automatic updates will not force frequent unplanned restarts

### 1.2 Windows Runtime

- [ ] virtualization is enabled
- [ ] WSL2 is installed and working
- [ ] Docker Desktop is installed
- [ ] Docker Desktop is configured for `Linux containers`
- [ ] Docker Desktop can start automatically with Windows

### 1.3 Network

- [ ] host is connected to the intended Seattle residential network
- [ ] direct upstream access from the host is healthy
- [ ] no unnecessary global VPN / proxy is enabled
- [ ] LAN IP is stable enough for maintenance use
- [ ] DNS and local network behavior are stable

### 1.4 Windows Security

- [ ] strong Windows password is in use
- [ ] MFA is enabled for important admin identities
- [ ] Windows Firewall is enabled
- [ ] no unnecessary public-facing local services are enabled
- [ ] no unrelated remote-assistance services are exposed

## 2. Deployment Directory

- [ ] fixed deployment directory created
- [ ] `.env` location decided
- [ ] `data` directory created
- [ ] `logs` directory created
- [ ] `backup` directory created

Record actual paths:

- compose file: `__________`
- env file: `__________`
- data dir: `__________`
- logs dir: `__________`
- backup dir: `__________`

## 3. Docker Runtime and Application

### 3.1 Compose Baseline

- [ ] `docker-compose.yml` is present
- [ ] `SESSION_SECRET` is set to a strong value
- [ ] `CRYPTO_SECRET` is set to a strong value
- [ ] database password is not using the default example value
- [ ] timezone is reviewed
- [ ] `restart: always` or equivalent restart behavior is configured

### 3.2 Containers

- [ ] `new-api` container starts successfully
- [ ] `postgres` container starts successfully
- [ ] `redis` container starts successfully
- [ ] persistent volumes are mounted
- [ ] logs are persisted outside ephemeral container state

### 3.3 Local Reachability

- [ ] `http://localhost:3000/api/status` is reachable
- [ ] service is not directly published to public internet
- [ ] no home router port forwarding exists for `new-api`

Record:

- local URL: `__________`
- compose project name: `__________`

## 4. `new-api` Operational Review

- [ ] rate limiting settings reviewed
- [ ] failure / circuit-break behavior reviewed
- [ ] channel auto-disable behavior reviewed
- [ ] default global system proxy is not enabled
- [ ] special proxy use stays at channel level
- [ ] no risky upstream header passthrough rules are configured by default

## 5. Cloudflare Tunnel

### 5.1 Installation

- [ ] `cloudflared` is installed
- [ ] installed version recorded
- [ ] `cloudflared tunnel login` completed

Record:

- `cloudflared` version: `__________`
- install path: `__________`

### 5.2 Tunnel Object

- [ ] dedicated Tunnel created
- [ ] Tunnel is not shared with unrelated projects
- [ ] Tunnel name is clear
- [ ] Tunnel ID is recorded

Record:

- tunnel name: `__________`
- tunnel ID: `__________`

### 5.3 Public Hostnames

- [ ] `admin` hostname points to `http://localhost:3000`
- [ ] `api` hostname points to `http://localhost:3000`
- [ ] no obsolete public hostname remains exposed

Record:

- admin route: `__________`
- api route: `__________`

### 5.4 Service Mode

- [ ] `cloudflared` installed as a Windows service
- [ ] service auto-start is enabled
- [ ] service restart behavior is acceptable
- [ ] logs or status can be inspected during troubleshooting

Record:

- service name: `__________`
- status/log inspection path: `__________`

## 6. Cloudflare Access

### 6.1 Applications

- [ ] `admin` self-hosted application created
- [ ] `api` self-hosted application created
- [ ] hostnames match the Tunnel configuration

### 6.2 `admin` Policy

- [ ] only intended identities are allowed
- [ ] MFA is required
- [ ] session duration is short enough
- [ ] no overly broad allow policy exists

### 6.3 `api` Policy

- [ ] API is not publicly open without Access
- [ ] access method matches actual usage
- [ ] if browser/manual use, identity login works
- [ ] if automation use, Service Token is configured

### 6.4 Service Token

If automation is used:

- [ ] Service Token exists
- [ ] Service Token is only used for the API app
- [ ] token is stored outside the repository
- [ ] token rotation method is documented

Record:

- token purpose: `__________`
- secret storage location: `__________`
- rotation cycle: `__________`

## 7. DNS and Origin Exposure

- [ ] DNS records are correct in Cloudflare
- [ ] `admin` hostname resolves as expected
- [ ] `api` hostname resolves as expected
- [ ] no DNS record exposes the residential public IP directly for the app
- [ ] no old testing record remains active
- [ ] origin is not directly publicly reachable

## 8. Header Safety Review

- [ ] no broad `HeaderOverride` passthrough is configured
- [ ] no risky `pass_headers` rule is configured
- [ ] no risky `copy_header` rule is configured
- [ ] no risky `{client_header:...}` usage is configured
- [ ] deployment aligns with the blocked ingress-header policy

## 9. External Validation

### 9.1 Admin Validation

- [ ] external browser can reach `admin` hostname
- [ ] unauthorized user is blocked
- [ ] MFA challenge works when expected
- [ ] admin UI loads correctly after authorization

### 9.2 API Validation

- [ ] external client can reach `api` hostname
- [ ] authorized request succeeds
- [ ] unauthorized request is blocked
- [ ] upstream relay behaves normally after Access

### 9.3 Recovery Validation

- [ ] after Windows reboot, Docker Desktop recovers
- [ ] after Windows reboot, `new-api` containers recover
- [ ] after Windows reboot, `cloudflared` recovers
- [ ] external access works again after reboot without manual terminal steps

## 10. Backup Operations

- [ ] private VPN is prepared for maintenance-only use
- [ ] maintenance path is documented
- [ ] emergency recovery path is documented
- [ ] `WARP` remains disabled by default
- [ ] conditions for enabling channel-level fallback are documented

Record:

- VPN solution: `__________`
- maintenance entry: `__________`
- emergency procedure location: `__________`

## 11. First 7 Days Observation

- [ ] admin hostname stability observed
- [ ] API hostname stability observed
- [ ] upstream success rate observed
- [ ] first-byte latency observed
- [ ] timeout rate observed
- [ ] circuit-break events observed
- [ ] channel auto-disable events observed
- [ ] host uptime observed

## 12. Change Log Discipline

For each important change, record:

- date
- change content
- reason
- affected scope
- rollback method

Suggested file names:

- `DEPLOYMENT_CHANGELOG.md`
- `RUNBOOK_PRIVATE.md`
