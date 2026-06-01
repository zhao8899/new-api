# 统一 AI API 网关最小化生产优化开发计划

## 1. 计划来源

本开发计划基于：

- `docs/security/AI_GATEWAY_MINIMAL_PRODUCTION_OPTIMIZATION_PLAN.md`
- 当前 `new-api` 架构：Go + Gin + GORM 后端、React 前端、多渠道 Relay、用户与计费系统、Redis 可选缓存

目标是把优化方案拆成可执行的开发批次、任务清单、验收门禁和回滚策略，便于按 PR 推进。

## 2. 开发目标

第一阶段不追求完整金融级网关，而是在不推翻现有架构的前提下完成最小化生产改造：

1. 消除高危默认行为和敏感信息泄露风险。
2. 建立统一模型和 Provider 治理基础。
3. 增强上游错误分类、重试、熔断和渠道降权能力。
4. 增加 RPM / TPM 基础限流，减少上游 429 和成本放大。
5. 建立请求追踪和管理审计，提升故障定位与追责能力。

## 3. 开发节奏

建议按 6 个开发批次推进，每个批次 1 周左右。若人力不足，可按 PR 顺序串行执行；若有多人协作，可按依赖图并行。

| 批次 | 周期 | 主题 | 主要 PR | 阶段目标 |
| --- | --- | --- | --- | --- |
| Sprint 1 | 第 1 周 | 安全基线 | PR-1、PR-2、PR-3 | 生产可安全启动，敏感信息不裸露，高危接口有二次验证 |
| Sprint 2 | 第 2 周 | 模型治理基础 | PR-4、PR-5 | 建立 Model Registry 和 Provider 分类，不破坏现有调用 |
| Sprint 3 | 第 3 周 | 错误与路由稳定性 | PR-6、PR-7、PR-8 | 上游错误可分类，渠道可熔断，重试受控 |
| Sprint 4 | 第 4 周 | 额度治理 | PR-9 | 网关侧具备 RPM / TPM 基础削峰能力 |
| Sprint 5 | 第 5 周 | 可观测与审计 | PR-10、PR-11 | 请求可追踪，管理操作可审计 |
| Sprint 6 | 第 6 周 | 发布固化 | PR-12 | 完成上线 checklist、回滚方案、冒烟和压测说明 |

## 4. 分支与提交策略

建议从 `main` 拉短分支，每个 PR 独立交付。

分支命名：

```text
security/production-startup-baseline
security/log-redaction
security/critical-route-verification
gateway/model-registry
gateway/provider-registry
gateway/provider-error-classification
gateway/channel-circuit-breaker
gateway/retry-budget
gateway/rpm-tpm-rate-limit
observability/request-trace
audit/audit-event
docs/production-release-gate
```

提交格式：

```text
feat(scope): short description
fix(scope): short description
docs(scope): short description
test(scope): short description
```

每个 PR 必须包含：

- 背景和目标
- 实现摘要
- 数据库兼容性说明
- 测试结果
- 风险和回滚方式

## 5. Sprint 1：安全基线

### PR-1：安全启动与初始化基线

任务：

- 梳理当前初始化流程：`model/main.go`、`controller/setup.go`、`constant.Setup`。
- 移除无用户时自动创建 `root / 123456` 的逻辑。
- 增加一次性 setup token 校验。
- 增加 `NEW_API_SECURITY_MODE=production`。
- 扩展生产启动检查：`SESSION_SECRET`、`CRYPTO_SECRET`、`SQL_DSN`、`REDIS_CONN_STRING`。
- 保持已有部署兼容：已有 root 用户时不强制重新初始化。

建议实现顺序：

1. 为 `common.ValidateProductionSecurityConfig` 增加测试用例。
2. 新增 setup token 读取与校验函数。
3. 修改 root 初始化逻辑。
4. 修改 setup controller。
5. 补充迁移兼容测试。

验收：

- 新实例不会生成固定密码 root。
- 生产模式缺关键配置拒绝启动。
- setup token 初始化成功后失效。
- 已初始化实例不受影响。

回滚：

- 可通过配置关闭生产模式。
- 保留已有 setup 记录和 root 用户不变。

### PR-2：敏感信息脱敏

任务：

- 新增统一脱敏工具包，建议放在 `common/redact.go` 或 `pkg/security/redact.go`。
- 实现文本、header、JSON body 三类脱敏。
- 接入 logger、错误日志、渠道测试、支付回调、OAuth 配置等高风险日志路径。
- 增加单元测试覆盖常见 key 格式。

脱敏范围：

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `password`
- `secret`
- `token`
- `access_token`
- `refresh_token`
- `api_key`
- `client_secret`
- `channel_key`
- `sk-...`

验收：

- 测试日志中不出现完整 token、key、secret。
- 脱敏后保留少量前后缀，便于排障。
- JSON body 脱敏失败时 fail-safe 返回整体脱敏结果。

回滚：

- 脱敏工具默认启用，不建议关闭。
- 若误伤排障，可增加 debug-only 局部开关，但生产模式必须保持脱敏。

### PR-3：WebSocket Origin 白名单与敏感接口二次验证

任务：

- 增加 WebSocket Origin 白名单配置。
- 生产模式下禁止 `CheckOrigin` 无条件返回 `true`。
- 识别高风险管理接口。
- 对高风险接口接入现有 secure verification。
- 增加最近验证时间窗口，建议 5 分钟。

首批高风险接口：

- 查看 / 导出 channel key
- 查看 token key
- 修改支付配置
- 修改 OAuth / OIDC 配置
- 修改渠道 base URL
- 新增 / 删除 channel
- 修改模型倍率和价格
- 删除日志
- 修改系统安全配置

验收：

- 未命中 Origin 白名单的浏览器 WebSocket 请求被拒绝。
- 高风险接口未二次验证时返回明确错误。
- 最近 5 分钟已验证的管理员可访问高风险接口。

回滚：

- WebSocket allowlist 可配置为兼容已有域名。
- 二次验证可先以少量接口灰度启用。

## 6. Sprint 2：模型治理基础

### PR-4：Model Registry

任务：

- 新增 `model_registry` 数据模型和迁移。
- 定义 external model、provider、upstream model、protocol、capabilities。
- 提供查询函数：按 external model 获取治理信息。
- 在现有模型解析路径旁路接入 registry。
- 未命中 registry 时回退现有逻辑。

建议字段：

```text
external_model
provider
upstream_model
protocol
capabilities
context_window
max_output_tokens
enabled
priority
```

首批模型分类：

- `chatgpt-4o`、`gpt-4.1` -> OpenAI
- `claude-sonnet`、`claude-sonnet-code` -> Claude
- `gemini-2.5-pro`、`gemini-2.5-flash` -> Gemini
- `deepseek-chat`、`deepseek-reasoner` -> OpenAI-compatible
- `qwen-max`、`qwen-plus`、`qwen-coder` -> OpenAI-compatible / DashScope

验收：

- registry 命中时能返回 provider、protocol、upstream model。
- registry 未命中时现有调用不受影响。
- SQLite、MySQL、PostgreSQL migration 均兼容。

回滚：

- 增加开关关闭 registry 接入。
- 表保留，不影响旧逻辑。

### PR-5：Provider 分类与 Adapter 基础接口

任务：

- 定义轻量 Provider Adapter 接口。
- 定义 provider registry 或 provider metadata。
- 对现有 provider 做分类，不强制迁移所有实现。
- 将 OpenAI-compatible 供应商统一标识。
- 为后续错误分类、路由、限流提供 provider/protocol 元数据。

验收：

- 新增 provider 类型不需要修改多处硬编码。
- 主流 provider 均可归类到 protocol。
- 现有 Relay 路径保持兼容。

回滚：

- Adapter 分类作为 metadata 使用，不替换现有 relay 主路径。

## 7. Sprint 3：错误与路由稳定性

### PR-6：统一错误分类

任务：

- 新增 `ProviderError` 类型。
- 建立 HTTP 状态码、上游错误码、网络错误到统一错误类型的映射。
- 在 OpenAI-compatible、Claude、Gemini 主路径优先接入。
- 错误分类输出到日志和 request trace 预留字段。

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

验收：

- 401 / 403 不重试，标记为认证或权限错误。
- 429 标记为限流，可触发渠道降权。
- 5xx 标记为服务端错误，可触发重试和熔断。
- timeout 和 network error 可区分。

回滚：

- 保留旧错误返回格式。
- 统一错误分类先用于内部决策和日志，不强制改变客户端响应。

### PR-7：渠道健康与熔断

任务：

- 新增 `channel_health` 模型和查询/更新服务。
- 记录 success、failure、429、5xx、timeout、latency。
- 实现 health score 和 circuit state。
- 在渠道选择时过滤 cooldown / open circuit 渠道。
- 增加探测恢复策略。

熔断规则：

- 连续 5 次 401 / 403：禁用或标记待人工处理。
- 连续 3 次 429：cooldown 30 到 120 秒。
- 连续 5 次 5xx：open circuit 60 秒。
- p95 latency 超阈值：降权。

验收：

- 熔断渠道不会被正常流量选中。
- cooldown 到期后可探测恢复。
- 管理端或日志可看到健康状态变化。

回滚：

- 开关关闭自动熔断，仅保留观测。

### PR-8：Retry Budget

任务：

- 为每次请求增加 retry budget。
- 明确不同错误类型下的重试策略。
- 记录 attempted channels、retry count、fallback used。
- 防止流式响应已输出后自动重试。
- 防止非幂等任务自动重试。

验收：

- 单请求默认最多重试 1 次。
- 401 / 403 不重试。
- 429 不在同渠道立即重试。
- 流式已输出后不自动换渠道。
- 任务型请求默认不自动重试。

回滚：

- 保留现有重试路径，新增预算开关可关闭。

## 8. Sprint 4：RPM / TPM 基础额度治理

### PR-9：Redis RPM / TPM 限流

任务：

- 定义限流 scope：provider、model、channel、group、token、user。
- 实现 RPM token bucket。
- 实现 TPM 估算和预占。
- 响应后按 actual usage 修正。
- 命中限流时返回统一 429 和 `Retry-After`。
- Redis 不可用时按配置 fail-open 或 fail-closed。

建议 Redis key：

```text
rate_limit:{provider}:{model}:{channel}:rpm
rate_limit:{provider}:{model}:{channel}:tpm
rate_limit:{provider}:{model}:{group}:rpm
rate_limit:{provider}:{model}:{token}:rpm
rate_limit:{provider}:{model}:{user}:rpm
```

验收：

- 可按 provider / model / channel 限制 RPM。
- 可按 provider / model / channel 限制 TPM。
- 并发请求下不会明显超发。
- 命中限流返回 `Retry-After`。

回滚：

- 限流策略默认 observe mode。
- 可按 scope 关闭阻断。

## 9. Sprint 5：可观测与审计

### PR-10：Request Trace

任务：

- 生成或透传 `request_id`、`trace_id`。
- 在 relayInfo 或上下文中贯穿 trace 字段。
- 记录 external model、provider、channel、latency、retry、fallback、tokens、cost、error type。
- 扩展日志查询字段。

验收：

- 任意请求可通过 request id 定位完整链路。
- 失败请求能看到错误类型、渠道、是否重试。
- 日志字段不包含未脱敏敏感信息。

回滚：

- 新字段只增不删，旧日志查询保持兼容。

### PR-11：Audit Event

任务：

- 新增 `audit_event` 表。
- 增加审计写入服务。
- 接入登录、token、channel、模型、价格、支付、OAuth、管理员、用户删除等关键路径。
- diff 内容必须脱敏。
- 审计查询接口必须有权限保护。

验收：

- 高风险管理操作都有审计记录。
- 审计记录可按 actor、action、resource、request_id 查询。
- 审计写入失败时，普通模型调用不受影响；敏感管理接口可配置 fail-closed。

回滚：

- 审计写入可暂时降级为日志输出。
- 表结构保留。

## 10. Sprint 6：发布固化

### PR-12：发布门禁、压测与回滚文档

任务：

- 编写生产配置 checklist。
- 编写安全验证 checklist。
- 编写关键接口冒烟测试清单。
- 编写 RPM / TPM 压测说明。
- 编写熔断和限流回滚方案。
- 将新增配置补充到 `.env.example` 或部署文档。

验收：

- 所有新增能力都有配置说明。
- 所有新增开关都有默认值和生产建议。
- 可以按 checklist 验证生产可上线状态。
- 可以按回滚文档关闭 registry、熔断、限流、审计阻断。

## 11. 全局验收门禁

每个 PR 合并前必须完成：

```bash
go test ./...
```

涉及前端时额外执行：

```bash
cd web/default
bun run build
```

涉及日志、安全、审计时额外检查：

```bash
rg -n "root / 123456|123456|Authorization: Bearer|sk-[A-Za-z0-9_-]{12,}|client_secret|channel_key" .
```

涉及数据库模型或迁移时必须确认：

- SQLite 可启动和迁移。
- MySQL 5.7.8+ 不使用不兼容语法。
- PostgreSQL 9.6+ 不依赖新版特性。
- raw SQL 中保留现有跨库 quoting 规则。

## 12. 风险控制

| 风险 | 控制措施 |
| --- | --- |
| 安全改动影响已有部署 | 生产模式才强制；旧实例保留兼容路径 |
| Model Registry 破坏旧模型映射 | 先 shadow mode，未命中回退旧逻辑 |
| 熔断误伤可用渠道 | 先 observe mode，再逐步阻断 |
| 限流误拒正常流量 | 分 scope 灰度，阈值先宽后紧 |
| 日志脱敏影响排障 | 保留安全前后缀和 request id |
| 审计写入影响主链路 | 普通调用 fail-open，高风险管理接口可 fail-closed |
| 测试已有失败阻塞开发 | 先建立已知失败清单，新增 PR 不扩大失败面 |

## 13. 交付物清单

第一阶段完成后应交付：

- 安全启动与 setup token 实现
- 统一脱敏工具和接入点
- WebSocket Origin 白名单
- 敏感接口二次验证
- Model Registry 数据模型和查询服务
- Provider 分类和轻量 Adapter 接口
- ProviderError 统一错误分类
- ChannelHealth 熔断和降权服务
- Retry Budget 实现
- Redis RPM / TPM 基础限流
- Request Trace 字段和查询能力
- Audit Event 模型和关键操作接入
- 生产发布 checklist
- 回滚说明和压测说明

## 14. 建议开始顺序

立即开始：

1. `PR-1 安全启动与初始化基线`
2. `PR-2 敏感信息脱敏`

这两个 PR 是后续所有生产化工作的安全底座，也最容易独立验收。

随后启动：

3. `PR-3 WebSocket Origin 白名单与敏感接口二次验证`
4. `PR-4 Model Registry`

当 Model Registry 和 Provider 分类稳定后，再进入错误分类、熔断、重试和限流。
