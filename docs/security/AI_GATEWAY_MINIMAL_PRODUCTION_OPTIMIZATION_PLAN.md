# 统一 AI API 网关最小化生产优化计划

## 1. 目标定位

本计划面向当前项目的渐进式生产优化，不推翻现有架构，不重写现有 Relay、渠道、计费、用户和管理后台主流程。

目标是在现有基础上，将系统从功能丰富的 AI API 中转平台，提升为生产可控的统一 AI API 网关 / 多模型接入与治理平台。

第一阶段优先补齐六项能力：

1. 生产安全基线
2. 多模型统一接入抽象
3. 高可用路由与熔断
4. RPM / TPM 基础额度治理
5. 请求可观测与定位
6. 管理操作审计

## 2. 非目标

以下能力属于完整金融级演进方向，不纳入第一阶段最小化改造范围：

- mTLS / DPoP
- KMS / Vault
- 完整 ABAC
- 不可篡改审计 Hash Chain
- 完整账本化计费
- OpenAPI / JSON Schema 全量校验
- SIEM 集成
- 灾备演练与合规报表
- 大规模重构 Provider / Relay 主架构

## 3. 当前关键风险

| 风险 | 当前证据 | 影响 | 优先级 |
| --- | --- | --- | --- |
| 默认 root 密码 | `model/main.go` 中无用户时创建 `root / 123456` | 生产环境可被接管 | P0 |
| WebSocket Origin 放行 | `controller/relay.go` 中 `CheckOrigin` 返回 `true` | 浏览器跨站滥用风险 | P0 |
| 生产密钥约束不足 | 已有生产检查，但未覆盖完整生产模式要求 | 会话、加密、审计不可控 | P0 |
| 敏感日志泄露 | 多处日志可能包含 token、key、secret、prompt | 密钥泄露、合规风险 | P0 |
| 多模型治理分散 | 模型、provider、protocol 缺少统一注册表 | 扩展成本高、路由不可控 | P1 |
| 上游错误未统一治理 | 401/403/429/5xx/timeout 策略分散 | 重试、退款、熔断行为不稳定 | P1 |
| 缺少请求全链路视图 | 日志难以完整串联 user、token、model、channel、cost | 故障定位慢 | P2 |

## 4. 总体实施原则

- 保留现有调用方式，避免破坏已有用户和渠道。
- 优先新增中间件、服务、表和旁路逻辑，少改核心 Relay 主路径。
- 每个 PR 保持小范围、可回滚、可单独验收。
- 所有涉及安全、计费、限流、审计的改动必须有测试。
- 所有数据库变更必须兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。
- JSON 编解码调用必须使用 `common/json.go` 中的封装函数。

## 5. PR 拆分与执行顺序

### PR-1：安全启动与初始化基线

目标：

- 移除固定 `root / 123456` 初始化逻辑。
- 增加一次性 setup token。
- 增加生产安全模式。

建议配置：

```bash
NEW_API_SECURITY_MODE=production
NEW_API_SETUP_TOKEN=...
SESSION_SECRET=...
CRYPTO_SECRET=...
SQL_DSN=...
REDIS_CONN_STRING=...
```

关键改动：

- `model/main.go`
- `controller/setup.go`
- `common/production_security.go`
- `common/init.go`

验收标准：

- 无用户首次启动时不会创建固定密码 root。
- 生产模式缺少 `SESSION_SECRET`、`CRYPTO_SECRET`、`SQL_DSN`、`REDIS_CONN_STRING` 时拒绝启动。
- setup token 只能在初始化阶段使用，初始化完成后失效。
- 旧部署升级路径不破坏已有 root 用户。

验证命令：

```bash
go test ./...
```

### PR-2：敏感信息脱敏

目标：

- 建立统一脱敏工具。
- 防止日志中泄露 token、key、secret、cookie、password、prompt 等敏感信息。

建议新增：

```go
func RedactSensitiveText(input string) string
func RedactHeaders(headers http.Header) http.Header
func RedactJSONBody(body []byte) []byte
```

至少覆盖字段：

- `authorization`
- `cookie`
- `set-cookie`
- `password`
- `key`
- `secret`
- `token`
- `access_token`
- `refresh_token`
- `api_key`
- `client_secret`
- `channel_key`

关键改动：

- `common/`
- `logger/`
- 涉及请求、响应、错误、渠道测试、支付、OAuth 的日志路径

验收标准：

- 日志中不出现完整 `Authorization`。
- 日志中不出现完整 `sk-` 类 token。
- 日志中不出现完整 channel key、OAuth secret、payment secret。
- 脱敏函数有单元测试覆盖常见格式。

验证命令：

```bash
go test ./...
rg -n "Authorization: Bearer|sk-[A-Za-z0-9_-]{12,}|password|client_secret" logs common logger controller relay service
```

### PR-3：WebSocket Origin 白名单与敏感接口二次验证

目标：

- WebSocket 不再默认允许任意 Origin。
- 高风险管理接口要求二次验证。

关键改动：

- `controller/relay.go`
- `middleware/auth.go`
- `controller/secure_verification.go`
- `router/api-router.go`
- 渠道、token、支付、OAuth、系统配置相关 controller

敏感接口范围：

- 查看 / 导出 channel key
- 批量查看 token key
- 修改支付配置
- 修改 OAuth / OIDC 配置
- 修改渠道 base URL
- 新增 / 删除上游渠道
- 修改模型倍率和价格
- 删除日志
- 修改系统安全配置

验收标准：

- 未配置 Origin 白名单时，生产模式下 WebSocket 拒绝浏览器跨站请求。
- 高风险接口要求最近 5 分钟内完成二次验证。
- 二次验证可复用密码确认、2FA、passkey 或现有 secure verification。

验证命令：

```bash
go test ./...
```

### PR-4：Model Registry

目标：

- 建立统一模型注册表。
- 固化 `external_model -> provider -> upstream_model -> protocol -> channel_pool` 映射。
- 兼容现有模型配置和渠道选择。

建议模型：

```go
type ModelRegistry struct {
    Id              int
    ExternalModel   string
    Provider        string
    UpstreamModel   string
    Protocol        string
    Capabilities    string
    ContextWindow   int
    MaxOutputTokens int
    Enabled         bool
    Priority        int
}
```

首批覆盖：

- ChatGPT / OpenAI
- OpenAI-compatible
- Claude / Claude Code
- Gemini
- DeepSeek
- Qwen
- Azure OpenAI
- Bedrock
- Ollama / local
- Midjourney / Suno task 类服务

关键改动：

- `model/`
- `setting/`
- `relay/common/`
- `middleware/distributor.go`
- 管理后台模型配置接口

验收标准：

- 业务侧可以使用统一 external model。
- 网关内部能解析 provider、protocol、upstream model。
- 未注册模型按现有逻辑兼容处理，避免破坏存量配置。

验证命令：

```bash
go test ./...
```

### PR-5：Provider 分类与 Adapter 基础接口

目标：

- 不重写所有 provider。
- 先建立统一 Provider 分类和轻量 Adapter 接口。
- 避免后续供应商逻辑继续堆入大分支。

建议接口：

```go
type ProviderAdapter interface {
    Name() string
    Protocol() string
    ValidateRequest(ctx context.Context, req any) error
    ConvertRequest(ctx context.Context, req any) (*http.Request, error)
    ParseResponse(ctx context.Context, resp *http.Response) (*UnifiedResponse, error)
    ParseError(ctx context.Context, resp *http.Response) (*ProviderError, error)
}
```

协议分类：

| Provider | Protocol |
| --- | --- |
| OpenAI / ChatGPT | `openai`, `responses` |
| Claude / Claude Code | `claude` |
| Gemini | `gemini` |
| DeepSeek | `openai-compatible` |
| Qwen | `openai-compatible`, `dashscope` |
| Azure OpenAI | `azure-openai` |
| Bedrock | `bedrock` |
| Ollama / local | `openai-compatible`, `local` |
| Midjourney / Suno | `task` |

验收标准：

- 新增 provider 分类不需要改动多个核心分支。
- OpenAI-compatible 供应商可共享通用路径。
- Claude、Gemini、task 类服务保留独立协议能力。

验证命令：

```bash
go test ./...
```

### PR-6：统一错误分类

目标：

- 统一不同上游的错误类型。
- 用错误分类驱动重试、切换、熔断和退款策略。

错误类型：

- `AUTH_ERROR`
- `RATE_LIMIT`
- `SERVER_ERROR`
- `TIMEOUT`
- `BAD_REQUEST`
- `CONTENT_FILTER`
- `MODEL_NOT_FOUND`
- `INSUFFICIENT_QUOTA`
- `NETWORK_ERROR`

建议模型：

```go
type ProviderError struct {
    Type                 string
    Provider             string
    StatusCode           int
    UpstreamCode         string
    MessageRedacted      string
    Retryable            bool
    Switchable           bool
    CircuitBreakerSignal bool
    RefundSuggested      bool
}
```

验收标准：

- 401 / 403 归类为 `AUTH_ERROR`。
- 429 归类为 `RATE_LIMIT`。
- 5xx 归类为 `SERVER_ERROR`。
- 超时和网络错误可区分。
- 错误日志只记录脱敏信息。

验证命令：

```bash
go test ./...
```

### PR-7：渠道健康与熔断

目标：

- 为每个渠道维护健康状态。
- 支持 429、5xx、timeout 自动降权和冷却。

建议表：

```text
channel_health
  channel_id
  provider
  model
  success_count
  failure_count
  rate_limit_count
  timeout_count
  p95_latency
  health_score
  circuit_state
  cooldown_until
  updated_at
```

状态：

- `healthy`
- `degraded`
- `cooldown`
- `open_circuit`
- `disabled`

基础规则：

- 连续 5 次 401 / 403：禁用渠道，等待人工处理。
- 连续 3 次 429：冷却 30 到 120 秒。
- 连续 5 次 5xx：熔断 60 秒。
- p95 延迟超过阈值：降权。
- 恢复后先放少量探测流量。

验收标准：

- 熔断渠道不会被正常调度选中。
- cooldown 到期后可进入探测。
- 渠道健康状态可被日志和管理接口查看。

验证命令：

```bash
go test ./...
```

### PR-8：Retry Budget

目标：

- 防止一次用户请求放大为多次上游请求。
- 明确不同错误的重试规则。

规则：

- 默认最多重试 1 次。
- 401 / 403 不重试。
- 429 不在同一渠道立即重试。
- 5xx 可换渠道重试。
- timeout 可换渠道重试。
- 流式响应已输出内容后不自动重试。
- 非幂等任务不自动重试。

建议 trace 字段：

- `retry_count`
- `fallback_used`
- `attempted_channels`
- `last_error_type`

验收标准：

- 单请求 retry 次数受控。
- 流式已输出后不会自动重新请求另一个渠道。
- 任务型请求默认不自动重试，除非显式幂等。

验证命令：

```bash
go test ./...
```

### PR-9：RPM / TPM 基础限流

目标：

- 在网关侧先做削峰，减少上游 429。
- 支持 RPM 和 TPM 双维度基础限流。

Redis key 建议：

```text
rate_limit:{provider}:{model}:{channel}:rpm
rate_limit:{provider}:{model}:{channel}:tpm
rate_limit:{provider}:{model}:{group}:rpm
rate_limit:{provider}:{model}:{token}:rpm
rate_limit:{provider}:{model}:{user}:rpm
```

请求前：

- 估算 input tokens。
- 结合 max output tokens 估算 TPM。
- 预占 RPM 和 TPM。
- 不足时本地返回 429 和 `Retry-After`。

响应后：

- 使用真实 usage 修正 TPM。
- 上游未执行或失败时释放部分预占。

验收标准：

- provider / model / channel 维度可限 RPM。
- provider / model / channel 维度可限 TPM。
- 限流命中返回统一错误模型和 `Retry-After`。
- Redis 不可用时按配置 fail-open 或 fail-closed。

验证命令：

```bash
go test ./...
```

### PR-10：Request Trace

目标：

- 每次请求都能完整定位调用路径、上游渠道、错误和成本。

建议字段：

```text
request_id
trace_id
user_id
token_id
group
external_model
internal_model
provider
channel_id
upstream_model
retry_count
fallback_used
status_code
upstream_status_code
latency_ms
prompt_tokens
completion_tokens
total_tokens
estimated_cost
actual_cost
error_type
error_message_redacted
created_at
```

关键改动：

- `model/log.go`
- `controller/relay.go`
- `relay/common/relay_info.go`
- `logger/`
- 管理后台日志查询接口

验收标准：

- 任意请求可以通过 `request_id` 串联用户、token、模型、渠道、错误、计费。
- 失败请求能看到 provider、channel、error type、是否重试。
- 日志不包含未脱敏敏感信息。

验证命令：

```bash
go test ./...
```

### PR-11：Audit Event

目标：

- 记录关键管理操作审计。
- 先做结构化审计，后续再升级不可篡改 hash chain。

建议表：

```text
audit_event
  id
  actor_id
  actor_role
  action
  resource_type
  resource_id
  source_ip
  request_id
  result
  diff_redacted
  created_at
```

第一阶段覆盖：

- 登录成功 / 失败
- 创建 token
- 查看 token key
- 查看 channel key
- 新增 / 修改 / 删除 channel
- 修改模型配置
- 修改价格倍率
- 修改支付配置
- 修改 OAuth / OIDC 配置
- 创建管理员
- 删除用户

验收标准：

- 高风险操作都有审计记录。
- 审计 diff 已脱敏。
- 审计查询接口有权限保护。

验证命令：

```bash
go test ./...
```

### PR-12：发布门禁与回滚固化

目标：

- 将前 11 个 PR 的安全、稳定性、限流、观测、审计要求固化成上线门禁。

交付内容：

- 生产配置 checklist
- 安全验证 checklist
- 压测脚本或压测说明
- 回滚方案
- 关键接口冒烟测试
- 运维排障说明

验收标准：

- `go test ./...` 通过。
- 前端相关改动时 `bun run build` 通过。
- 生产模式下安全门禁生效。
- 关键管理接口二次验证生效。
- 日志脱敏检查通过。
- 限流、熔断、重试策略有测试覆盖。

验证命令：

```bash
go test ./...
cd web/default
bun run build
```

## 6. 依赖图

```text
PR-1 安全启动
  -> PR-3 WebSocket / 敏感接口保护
  -> PR-12 发布门禁

PR-2 日志脱敏
  -> PR-6 统一错误分类
  -> PR-10 Request Trace
  -> PR-11 Audit Event

PR-4 Model Registry
  -> PR-5 Provider 分类
  -> PR-6 统一错误分类
  -> PR-7 渠道健康与熔断

PR-6 统一错误分类
  -> PR-7 渠道健康与熔断
  -> PR-8 Retry Budget
  -> PR-9 RPM / TPM 限流

PR-10 Request Trace
PR-11 Audit Event
  -> PR-12 发布门禁
```

可并行：

- PR-1 与 PR-2 可并行。
- PR-4 可在 PR-1 / PR-2 后并行启动。
- PR-10 的字段设计可提前做，但接入应等 PR-6 / PR-7 稳定后收口。

## 7. 测试策略

每个 PR 至少包含：

- 单元测试：核心函数、状态机、错误分类、脱敏、限流计算。
- 集成测试：涉及 controller、middleware、DB migration 的路径。
- 回归测试：保证已有 OpenAI-compatible、Claude、Gemini、任务型接口不破坏。

重点测试：

- 默认 root 不再创建。
- 生产模式缺密钥拒绝启动。
- WebSocket Origin 白名单。
- 敏感日志脱敏。
- 高风险接口二次验证。
- ProviderError 分类。
- 渠道 cooldown / circuit breaker。
- Retry Budget 不放大请求。
- RPM / TPM 本地限流。
- Request Trace 字段完整。
- Audit Event 记录和脱敏。

## 8. 发布与回滚策略

发布策略：

- 安全基线先灰度到测试环境。
- Model Registry 默认 shadow mode，不立即阻断未注册模型。
- 熔断和限流先以 observe mode 记录，不立即拒绝生产流量。
- 限流阈值按 provider / model / channel 分批开启。
- 审计先只写不阻断，再逐步接入敏感接口强制策略。

回滚策略：

- 每个新能力必须有配置开关。
- Model Registry 支持回退现有模型映射。
- 熔断支持关闭自动禁用，仅记录健康状态。
- RPM / TPM 限流支持 per-scope 关闭。
- 审计写入失败不得影响普通模型调用，但敏感管理接口可 fail-closed。

## 9. 成功标准

完成本计划后，系统应具备：

- 不存在固定默认管理员密码。
- 生产模式具备强制启动门禁。
- 日志中敏感信息默认脱敏。
- ChatGPT / OpenAI-compatible、Claude、Gemini、DeepSeek、Qwen 等可通过统一模型注册表治理。
- 上游 401 / 403 / 429 / 5xx / timeout 有统一分类。
- 渠道可根据错误率、限流率、延迟自动降权或熔断。
- 单请求重试次数受控，不放大上游成本。
- RPM / TPM 可在网关侧基础治理。
- 每次请求可定位 user、token、model、provider、channel、latency、tokens、cost、error。
- 高风险管理操作有二次验证和结构化审计。

## 10. 后续金融级演进

第一阶段完成后，再按以下方向演进：

1. KMS / Vault 与密钥版本管理
2. 不可篡改 Audit Hash Chain
3. BillingLedger 账本化计费
4. mTLS / DPoP / 企业 ClientCredential
5. RouteSecurityPolicy 与审批流
6. OpenAPI / JSON Schema 请求校验
7. SIEM / 告警 / SLO 报表
8. 灾备演练、压测报告和合规文档
