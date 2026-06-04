# Seattle Residential Deployment Manual

## 1. Purpose

This manual documents the recommended deployment pattern for running `new-api` on a Seattle residential Windows 11 laptop while allowing stable and secure remote usage from external devices.

The target architecture is:

`External Device -> Cloudflare Access -> Cloudflare Tunnel -> Windows 11 Host -> Docker Desktop (WSL2 / Linux containers) -> new-api -> Upstream AI Providers`

Default upstream egress should remain the Seattle residential network. `WARP` is reserved as a per-channel backup option and is not enabled globally by default.

## 2. Scope

This manual covers:

- host preparation for the Seattle laptop
- recommended Windows 11 runtime choice
- `new-api` deployment using Docker Desktop and Linux containers
- Cloudflare Tunnel and Access setup
- remote usage model
- backup operations path
- optional `WARP` fallback strategy
- header forwarding safety constraints
- validation and troubleshooting

This manual does not replace provider-specific API setup or local Windows administration policies.

## 3. Design Goals

- external devices can use `new-api` easily from any location
- upstream providers observe the Seattle residential egress by default
- the laptop is not directly exposed through public inbound ports
- the management surface is protected by identity-aware access control
- operational changes remain centralized on the Seattle laptop
- Windows remains only the host OS, while the application runtime stays aligned with the project's Linux container delivery model

## 4. Final Topology

### 4.1 Main Path

`Browser / client / script -> Access-protected domain -> Cloudflare Tunnel -> localhost service on Seattle laptop -> Docker Desktop -> new-api -> upstream provider`

This path is responsible for:

- admin panel access
- API calls
- stable daily usage from external locations

### 4.2 Backup Paths

- maintenance path: private VPN to the Seattle laptop
- upstream fallback path: per-channel `WARP` or proxy, only when a specific channel needs it

Neither backup path should be the default daily path.

## 5. Preconditions

Before deployment, confirm all of the following:

- the Seattle laptop uses a stable residential network
- direct upstream access from the laptop is already stable
- the laptop can stay online long-term
- you control a Cloudflare account and domain
- you can create Zero Trust / Access applications
- you can install Windows services on the laptop
- Windows 11 virtualization is enabled
- WSL2 is installed and working
- Docker Desktop is installed
- Docker Desktop is running in `Linux containers` mode

For the current project, the recommended Windows deployment model is:

- `Windows 11` as the long-running host
- `Docker Desktop + WSL2` as the runtime
- `new-api / PostgreSQL / Redis` as Linux containers

This is the most stable Windows deployment path for this project because:

- the project delivery model is Docker-first
- the repository already provides `Dockerfile` and `docker-compose.yml`
- source-based native deployment requires additional frontend build steps before backend compilation
- long-term operations, upgrades, rollback and recovery are simpler than native Windows deployment

If any of the above is not satisfied, fix the host prerequisites first.

## 6. Host Preparation

### 6.1 Power and Availability

The Seattle laptop must behave like a long-running edge host.

Required settings:

- keep the laptop plugged in
- disable sleep
- disable hibernation
- disable lid-close sleep behavior
- avoid update windows that force frequent unplanned restarts

Operational target:

- after reboot, core services recover automatically
- no interactive login should be required just to restore service availability

### 6.2 Network Stability

Recommended:

- prefer wired Ethernet when available
- if Wi-Fi is used, keep the laptop on a fixed home network
- assign a stable LAN IP address
- avoid enabling multiple global VPN / proxy / packet-capture tools at the same time

Not recommended:

- using the same laptop for heavy network experiments
- using the same laptop as both a high-change daily workstation and a stable production ingress host
- frequently switching hotspots or temporary networks

### 6.3 Windows Security Baseline

At minimum:

- use a strong Windows account password
- enable MFA on important management accounts
- keep Windows Firewall enabled
- avoid exposing unrelated local services publicly
- avoid unnecessary remote-assistance software

## 7. `new-api` Deployment Model

### 7.1 Fixed Deployment Directory

Use a fixed deployment directory on Windows, for example:

- `D:\srv\new-api\docker-compose.yml`
- `D:\srv\new-api\.env`
- `D:\srv\new-api\data\`
- `D:\srv\new-api\logs\`
- `D:\srv\new-api\backup\`

Do not mix logs, config, data and temporary files together.

### 7.2 Recommended Runtime Choice

For this project, the recommended Windows runtime model is:

`Windows 11 -> Docker Desktop -> WSL2 -> Linux containers -> new-api`

This is preferred over native Windows deployment because:

- the official deployment examples are Docker / Compose based
- the backend embeds frontend static files, so native source deployment requires a frontend build first
- native Windows deployment adds extra work for Bun, Go build, service wrapping, restart handling and logging

This manual therefore assumes `new-api` runs through Docker Compose.

### 7.3 Local Exposure Strategy

`new-api` should not be directly exposed to the LAN or public internet.

Recommended:

- keep the service reachable locally at `http://localhost:3000`
- publish it externally only through Cloudflare Tunnel
- do not configure router port forwarding for port `3000`

If you want separate external hostnames, keep the same local-only principle:

- `admin.example.com` -> `http://localhost:3000`
- `api.example.com` -> `http://localhost:3000`

The hostname split is for access control and operational clarity, not for exposing multiple public ports.

### 7.4 Service Management

For the recommended deployment model:

- Docker Desktop manages the Linux runtime
- container restart behavior is managed by Compose policies
- `cloudflared` should still run as a Windows service

Operational requirements:

- automatic startup on boot
- automatic restart on failure
- persistent logs
- no dependency on an open interactive terminal

Recommended:

- configure Docker Desktop to start with Windows
- use `restart: always` for the application containers
- install `cloudflared` as a Windows service

### 7.5 Internal `new-api` Settings

Keep these features enabled or reviewed:

- rate limiting
- 401 / 403 or critical error circuit breaking
- automatic channel disable rules
- channel-level proxy configuration only when required
- logging and monitoring

At the same time:

- do not enable a system-wide global proxy by default
- handle special network requirements at channel level
- do not forward client-origin headers to upstream providers unless explicitly required and reviewed

### 7.6 Docker Compose Recommendations

Use the repository `docker-compose.yml` as the base template and modify:

- PostgreSQL password
- `SESSION_SECRET`
- `CRYPTO_SECRET`
- `TZ`
- whether Redis / PostgreSQL are retained

Recommended services:

- `new-api`
- `postgres`
- `redis`

Before going live, verify locally:

- `http://localhost:3000/api/status` is reachable
- data survives container restart
- after Windows reboot, Docker Desktop and the containers recover automatically

## 8. Remote Access Strategy

### 8.1 Why `Tunnel + Access` Is the Main Path

For this use case, `Tunnel + Access` is preferred because:

- you need remote access to the application, not the whole home network
- the daily surface is the admin UI and HTTP API, not a full remote desktop workflow
- access can be controlled per application instead of by exposing raw domains
- no public inbound port needs to be opened on the home router

### 8.2 Why VPN Is Still Kept

VPN is still recommended as an operational backup because it is useful for:

- SSH / PowerShell maintenance
- remote desktop
- viewing logs
- restarting services
- fixing Tunnel / Access issues

Conclusion:

- daily usage: `Tunnel + Access`
- system maintenance: private VPN

## 9. Domain Plan

Use at least two hostnames:

- `admin.example.com`
- `api.example.com`

Responsibilities:

- `admin.example.com`: management panel only
- `api.example.com`: client / script / application API access

Do not:

- mix admin and API into one unrestricted public surface
- overload one hostname with unrelated public paths when separate policy control is needed

## 10. Cloudflare Tunnel Setup

### 10.1 Install `cloudflared`

Install `cloudflared` on the Seattle laptop.

Requirements:

- use a current stable build
- record the installed version
- update it periodically

After installation, verify:

- `cloudflared --version`
- `cloudflared tunnel login` runs successfully

### 10.2 Create a Dedicated Tunnel

Create one dedicated Tunnel for this deployment.

Do not reuse a mixed-purpose Tunnel for unrelated projects or hosts.

Recommended steps:

1. `cloudflared tunnel login`
2. `cloudflared tunnel create new-api-seattle`
3. record the Tunnel ID
4. bind DNS records to the tunnel

### 10.3 Bind Hostnames

Recommended mapping:

- `admin.example.com` -> `http://localhost:3000`
- `api.example.com` -> `http://localhost:3000`

If you temporarily want only one unified entrypoint, start with:

- `api.example.com` -> `http://localhost:3000`

Recommended DNS route commands:

```powershell
cloudflared tunnel route dns new-api-seattle admin.example.com
cloudflared tunnel route dns new-api-seattle api.example.com
```

Recommended `config.yml`:

```yaml
tunnel: <Tunnel-ID>
credentials-file: C:\Users\<your-user>\.cloudflared\<Tunnel-ID>.json

ingress:
  - hostname: admin.example.com
    service: http://localhost:3000
  - hostname: api.example.com
    service: http://localhost:3000
  - service: http_status:404
```

Foreground test:

```powershell
cloudflared tunnel run new-api-seattle
```

Once hostname access works, switch to service mode.

### 10.4 Run Tunnel as a Windows Service

`cloudflared` should run as a Windows service.

Requirements:

- auto-start
- auto-restart
- persistent logs

Recommended steps:

1. `cloudflared service install`
2. verify the service in Windows Service Manager or PowerShell
3. verify that the service auto-recovers after Windows reboot

### 10.5 Tunnel Security Principle

Tunnel is the only public publishing entrypoint.

Make sure:

- no home-router port forwarding is added for `new-api`
- the backend port is not exposed publicly
- the API is not directly published to the public internet

## 11. Cloudflare Access Configuration

### 11.1 Create Two Self-Hosted Applications

Create:

- `admin.example.com`
- `api.example.com`

### 11.2 `admin` Application Policy

Recommended:

- allow only your own identity
- require MFA
- use a shorter session duration
- avoid broad allow rules

The principle is: the admin surface should stay narrow and strict.

### 11.3 `api` Application Policy

Choose based on usage:

- browser / manual use: identity login protection
- scripts / applications / automation: Service Token

Recommended:

- do not leave `api.example.com` publicly open
- keep Access protection even for personal usage

For automated clients using Service Token, send:

- `CF-Access-Client-Id`
- `CF-Access-Client-Secret`

### 11.4 Origin Protection Principle

Recommended model:

- Access authenticates the caller first
- Tunnel forwards traffic to localhost
- `new-api` remains hidden from direct public origin access

This avoids the common mistake where the domain is protected but the origin is still directly reachable.

## 12. Upstream Egress Strategy

### 12.1 Default Egress

By default, `new-api` should reach upstream providers through:

`new-api -> Seattle residential network -> upstream providers`

This means:

- upstream sees the Seattle residential egress
- fewer middle layers are introduced
- troubleshooting stays simpler
- the design stays aligned with the residential-host assumption

### 12.2 Why `WARP` Is Not Enabled by Default

If direct upstream access from the Seattle laptop is already stable, global `WARP` is not the preferred default.

Reasons:

- shorter path
- fewer variables
- fewer failure points
- cleaner upstream egress identity

### 12.3 Proper Placement of `WARP`

`WARP` should not be placed in the main path between external users and the Seattle host.

Its correct role is:

- backup egress for specific channels
- targeted fallback when a specific upstream path is unstable

Meaning:

- main path: no `WARP`
- abnormal specific channel: optional `WARP`

### 12.4 When to Enable `WARP`

Only consider enabling `WARP` for a specific channel when:

- one upstream repeatedly times out
- one channel shows persistent first-byte or connectivity instability
- one upstream needs a dedicated backup egress path

Do not enable it globally just because it might be theoretically more stable.

## 13. Channel-Level Proxy Strategy

All proxies and special network paths in `new-api` should stay at channel level, not system-global level.

Recommended principles:

- no global system proxy by default
- configure a proxy only inside the specific channel that needs it
- isolate abnormal channels instead of changing the host's global egress model

Benefits:

- clearer fault isolation
- one configuration change does not affect the whole system
- direct path and fallback path remain easier to compare

## 14. Upstream-Visible Information and Header Safety

### 14.1 What Upstream Should Normally See

Under the recommended architecture, upstream providers should normally see only:

- the Seattle residential egress IP
- headers intentionally constructed by `new-api`
- business fields required by the upstream protocol

### 14.2 What Upstream Should Not See by Default

The following should not be leaked to upstream by default:

- the external device's real public IP
- the external device's office or travel network identity
- Cloudflare Access identity headers
- proxy-chain source headers

### 14.3 Existing Unified Blocking

The project already blocks the following from being forwarded upstream at the unified outbound layer:

- `X-Forwarded-*`
- `Forwarded`
- `X-Real-IP`
- `CF-Connecting-IP`
- `True-Client-IP`
- `CF-Access-*`

Related record:

- [HEADER_FORWARDING_OPTIMIZATION_RECORD.md](../security/HEADER_FORWARDING_OPTIMIZATION_RECORD.md)

### 14.4 Configuration-Layer Recommendations

Even if the code already blocks these headers, avoid overly broad passthrough at configuration level:

- broad `HeaderOverride` passthrough
- generic regex passthrough
- `{client_header:...}`
- `pass_headers`
- `copy_header`

Only pass specific headers when a provider explicitly requires them and the behavior has been reviewed.

## 15. Maintenance Backup Path

Keep a private VPN as the maintenance-only backup route.

Use it for:

- logging into the host
- viewing logs
- manually restarting services
- repairing Tunnel
- repairing Access
- system-level maintenance

Do not use the VPN as the main daily path to access `new-api`.

## 16. Recommended Deployment Order

### Stage 1: Stabilize the Host

1. adjust power settings
2. disable sleep and hibernation
3. stabilize the residential network connection
4. confirm direct upstream connectivity
5. complete Windows security baseline settings

### Stage 2: Stabilize the Docker Runtime

1. install and start Docker Desktop
2. confirm Docker Desktop uses `Linux containers`
3. place `docker-compose.yml`, `.env`, data and log directories into a fixed deployment path
4. start `new-api / postgres / redis`
5. verify `http://localhost:3000/api/status`
6. verify automatic recovery after Windows reboot

### Stage 3: Publish Through Tunnel

1. install `cloudflared`
2. create a dedicated Tunnel
3. bind `admin.example.com`
4. bind `api.example.com`
5. install `cloudflared` as a Windows service

### Stage 4: Lock Down with Access

1. create the `admin` Access application
2. create the `api` Access application
3. configure identity restrictions
4. configure MFA
5. if automation is needed, configure Service Token

### Stage 5: Validate

1. access the admin hostname externally
2. call the API hostname externally
3. verify unauthorized users cannot access
4. verify upstream still uses the Seattle residential egress by default

### Stage 6: Prepare Backup Paths

1. prepare the private VPN
2. prepare system-maintenance access
3. keep `WARP` disabled by default
4. enable fallback egress only for channels that actually need it

## 17. Acceptance Criteria

After deployment, at minimum all of the following should be true:

- external browsers can reliably open `admin.example.com`
- external clients can reliably call `api.example.com`
- unauthorized users cannot access the service
- after host reboot, Docker Desktop, the `new-api` containers and `cloudflared` all recover automatically
- the home network does not expose a public inbound port for `new-api`
- the default upstream egress remains the residential network
- sensitive source headers are not forwarded upstream

## 18. Daily Checklist

Keep watching:

- whether the laptop is online
- Docker Desktop status
- `new-api` container status
- `cloudflared` service status
- admin hostname reachability
- API hostname reachability
- upstream success rate
- first-byte latency
- timeout rate
- circuit-breaker triggers
- channel auto-disable events

## 19. Troubleshooting

### 19.1 Cannot Open the Admin Hostname Externally

Check in order:

- whether `cloudflared` is running
- whether Access policy is blocking by mistake
- whether the local backend is still listening on localhost
- whether the hostname mapping is correct

### 19.2 Cannot Open the API Hostname Externally

Check in order:

- whether Access configuration is correct
- if Service Token is used, whether the token is valid
- whether the local service is reachable on the expected port
- whether Windows Firewall is interfering with the local routing or service process

### 19.3 Upstream Timeout or Response Becomes Unstable

Check in order:

- whether direct upstream access from the Seattle laptop is still healthy
- whether recent channel configuration changed
- whether only one provider is affected
- whether that single channel needs a fallback egress path

If only one provider is abnormal, fix that provider's channel first instead of changing the host-wide network model.

### 19.4 Suspected Leakage of External Client Information Upstream

Check:

- whether `HeaderOverride` is being used
- whether `pass_headers` is configured
- whether `copy_header` is configured
- whether `{client_header:...}` is configured
- whether any custom adaptor changed outbound header behavior

## 20. Change Management

After each important change, record:

- date
- what changed
- why it changed
- affected scope
- rollback method

Prioritize recording:

- host network changes
- Tunnel changes
- Access changes
- `new-api` service configuration changes
- channel proxy changes
- `WARP` fallback changes

## 21. Related Documentation

Maintain this manual together with:

- [HEADER_FORWARDING_OPTIMIZATION_RECORD.md](../security/HEADER_FORWARDING_OPTIMIZATION_RECORD.md)
- local deployment notes
- a private environment parameter checklist

The private checklist should be maintained separately and should include:

- real domains
- actual local ports
- service paths
- log directories
- backup VPN information
- maintenance contacts and recovery paths

## 22. Final Operating Principles

In long-term operation, this solution should always follow these principles:

- use the Seattle residential laptop as the stable host
- use `Tunnel + Access` as the daily ingress path
- use private VPN as the maintenance-only backup path
- keep residential direct egress as the default upstream path
- use channel-level isolation for abnormal networking needs
- minimize exposure surface
- keep documentation current enough to support recovery
