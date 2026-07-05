# 下一阶段企业级硬化 Goal 提示词

你正在 Nivora 仓库中工作。

这是一个超大生产级企业硬化与能力收口任务。目标不是继续做 MVP，不是继续扩散骨架，也不是只写文档。目标是把 Nivora 从 hardened beta-candidate foundation 推进到更接近 enterprise production-candidate 的前置状态。

Nivora 仍然不得声称 production-ready。所有输出必须诚实标注 foundation、partial、experimental、beta-candidate、not production-ready。

## 项目背景

Nivora 是开源 DevOps delivery control plane，后端主语言 Go，架构是 modular monolith，包含：

- nivora-server
- nivora-worker
- nivora-runner
- nivora CLI
- nivora-mcp
- experimental web console

Nivora 的核心价值不是替代 Jenkins、Argo CD、Kubernetes、Harbor、Trivy、Vault 或云厂商，而是统一交付意图、运行状态、artifact 身份、策略、审批、日志、事件、审计、证据包、runner 状态和 AI/MCP 安全读取能力。

## 必须遵守

- 读取并遵守 `AGENTS.md`。
- 遵守 `docs/architecture/architecture-contract.md` 和 `docs/architecture/module-boundaries.md`。
- Domain 层不得依赖 HTTP、数据库、Kubernetes、cloud SDK、Argo CD、日志或 telemetry 包。
- Usecase 依赖 ports/interfaces，不直接绑定 concrete adapters。
- Adapter/infra 承担外部系统和技术实现。
- 不添加云厂商真实部署集成。
- 不默认启用 shell executor、Kubernetes apply、Argo sync、host remote deploy、Git push、rollback execution。
- 不提交 secret、token、kubeconfig、私钥、密码或看起来真实的假凭据。
- 不让 secret/token/hash 通过普通 API、CLI、MCP、audit、event、log 泄漏。
- 不声称 production-ready。
- 不把 remote MCP 作为广泛可用生产入口。
- 不把 shell executor 描述为 sandbox。
- 每次 commit 后必须 push `origin main`。
- 如果 GPG/1Password 签名失败，可对当次提交使用 `git -c commit.gpgsign=false commit`。

## 本轮必须完成的六大方向

每个方向都要优先做真实功能闭环。每个闭环至少包含：

- 实际代码实现
- API 或 CLI 或 MCP 可操作入口
- 正向测试
- 负向测试
- 安全/权限/边界测试
- OpenAPI/AsyncAPI 更新，如 API/event 变化
- 文档和 status 更新
- examples 或 smoke 脚本，如适用
- 不扩大危险默认行为
- 本地验证通过
- push 后 GitHub CI 通过

## 方向一：Runner Sandbox / Isolated Executor

目标：降低 shell runner 作为生产最大风险面的暴露程度。

优先能力：

- runner isolation profile 明确化：`local-dev`、`shell-hardened`、`container-isolated`、`external-required`
- production 配置默认拒绝不安全 profile
- executor capability negotiation：runner 声明 `shell`、`container`、`kubernetes-job`、`webhook` 等能力
- job claim 必须匹配 executor type、runner group、labels、environment/project scope、max concurrency
- shell executor 继续标记为非 sandbox
- container executor 或 external executor foundation 只能在显式启用时使用
- logs/status/cancel 仍然只能由 owning runner 修改
- runner token rotate/revoke 后旧 token 不能 claim、heartbeat、append logs、update status
- 增强 docs/security/runner-trust-boundary.md 和 docs/operations/runner-security.md

非目标：

- 不实现完整容器平台。
- 不默认挂载 Docker socket。
- 不执行不可信 workload。
- 不把 shell executor 宣称为安全沙箱。

验收：

- 生产配置拒绝不安全 runner profile。
- runner claim 能证明 capability/scope/concurrency 生效。
- 无关 runner 不能修改 job。
- CLI/API/docs/status 更新。
- 测试覆盖 token、scope、executor capability、unsafe profile。

## 方向二：Kubernetes CD Production Hardening

目标：让 Kubernetes YAML 从 experimental guarded foundation 更接近 beta-grade，但仍不声称 GA。

优先能力：

- server-side dry-run 与 apply 清晰分离
- apply 必须 `confirm=true` 且 target/config allowApply=true
- namespace/context/cluster safety policy
- namespace allowlist / denylist foundation
- manifest size/resource count limits
- resource inventory persistence and query
- health/rollout watch 对 Deployment、StatefulSet、DaemonSet、Job 明确支持边界
- rollback 走 manifest restore strategy，默认不 prune/delete
- pruning policy 只做 plan/preview，默认不执行
- API/CLI 显示 unsafe reason 和 required confirmation

非目标：

- 不实现 Helm/Kustomize。
- 不实现 operator。
- 不在 CI 依赖真实集群。
- 不默认 delete/prune。

验收：

- fake Kubernetes adapter 覆盖 dry-run/apply/rollback/namespace safety。
- apply/rollback 缺 confirm 或 allowApply 时失败。
- OpenAPI/CLI/docs/status 更新。

## 方向三：Remote MCP Read-only Safety

目标：让 remote MCP 从 experimental opt-in foundation 推进到可控 beta 预备，但仍默认关闭。

优先能力：

- remote MCP 只读/plan-only，action tool 继续 blocked
- bearer/service-account/OIDC placeholder subject 必须有 scope
- runner token 必须拒绝
- per-subject rate limit，request timeout，request/response cap
- list-like resources 支持 limit/offset
- tenant ownership checks for PipelineRun、DeploymentRun、ReleaseExecution、Artifact、SecurityScan、PolicyResult、Evidence、Audit、Runner summary
- audit records include remote client/request/correlation metadata where available
- unknown resource/tool structured errors
- prompt injection corpus 增强

非目标：

- 不开放 MCP action。
- 不读 secret。
- 不 approve/reject/apply/sync/rollback。
- 不把 remote MCP 广泛暴露为生产接口。

验收：

- remote MCP disabled by default。
- remote MCP auth/scope/rate/size/timeout tests。
- cross-tenant resource tests fail closed。
- MCP_PERMISSION_MATRIX、MCP_TENANT_SCOPE_REVIEW、status docs 更新。

## 方向四：Install / Backup / Restore / Upgrade Drills

目标：把安装和恢复从“文档和可选脚本”推进到更可重复的工程验证。

优先能力：

- Helm production profile static safety test
- Compose production-like profile smoke
- migration up/down integration test
- release-to-release migration compatibility smoke
- backup PostgreSQL、object store、config、secret metadata 的 drill script
- restore drill script 验证关键表、audit chain、evidence bundle、runtime recovery
- production doctor 增加 live database/outbox/audit-chain check 的显式模式
- CI 或可选 profile 记录 skip reason，不假装通过

非目标：

- 不添加 operator。
- 不添加云备份集成。
- 不要求本地 kind/cloud。

验收：

- `make smoke-backup-restore` 或等价目标可运行/可跳过并说明原因。
- GitHub postgres-integration 覆盖 migration/store/recovery/audit。
- docs/operations/backup-restore.md、install docs、status 更新。

## 方向五：Web Console Core Workflows

目标：让 web console 能展示核心控制平面，不做完整 UI 产品，不扩大后端语义。

优先页面：

- Dashboard
- PipelineRun detail
- DeploymentRun detail
- ReleaseExecution detail
- Artifact / Policy / Security summary
- Runner summary
- Audit / Evidence view
- MCP safety/status view if API already exists

要求：

- 只消费已有 API。
- 不造假生产数据。
- 清晰 loading/error/empty states。
- API base URL 由 env 配置。
- 不硬编码 token/secret。
- 不添加重型 UI 框架。
- React 代码遵守 react-best-practices：避免不必要 rerender、避免 barrel import、避免巨大同步循环、避免 waterfall。
- 用 Playwright 或现有 web verification 做真实页面 smoke。

非目标：

- 不做完整设计系统。
- 不做复杂 auth login UI，除非已有后端契约足够。
- 不新增后端产品 API。

验收：

- `make verify-web` 通过。
- 主要页面能访问且不会全是 Failed to fetch。
- docs/dev/web-console.md 和 README/status 更新。

## 方向六：External Integration Hardening by Boundary

目标：逐个把外部集成从 skeleton/foundation 推进到 beta-grade boundary，不同时铺开所有真实集成。

优先顺序：

1. Artifact registry / OCI / Harbor-compatible through generic OCI API
2. Argo CD status read and guarded sync safety
3. Git provider metadata/read-only validation
4. Secret provider validation contracts
5. Scanner adapter contracts
6. Cloud inventory remains skeleton unless explicitly scoped later

要求：

- 每个外部集成都必须有 fake/noop adapter 支持 CI。
- 网络访问必须 optional。
- CredentialRef/SecretRef 只传引用，不泄漏值。
- insecure/local endpoint 必须显式配置。
- docs 必须明确哪些是 real、fake、noop、skeleton。

非目标：

- 不实现云部署。
- 不实现 Harbor/Nexus/JFrog 管理 API。
- 不实现完整 Argo automation。
- 不提交凭据。

验收：

- 至少一个外部集成边界从 foundation 推进到 beta-grade boundary。
- fake adapter 测试覆盖成功/失败/credential redaction。
- OpenAPI/CLI/docs/status 更新。

## 总体验证

每个可提交闭环都必须运行：

```bash
go mod tidy
make fmt-check
go test ./...
go vet ./...
go build ./cmd/nivora-server
go build ./cmd/nivora-worker
go build ./cmd/nivora-runner
go build ./cmd/nivora
go build ./cmd/nivora-mcp
./scripts/verify-architecture.sh
./scripts/verify-no-secrets.sh
make verify-mcp
make verify-enterprise-readiness
make verify
git diff --check
```

如果存在且环境可用：

```bash
make test-postgres-integration
make verify-runtime-recovery
```

如果本地跳过 Postgres/Docker/Kubernetes/Helm/Playwright，必须记录原因。推送后必须查看 GitHub CI，尤其是 `postgres-integration`。

## 最终回复必须中文

最终回复必须包含：

1. 完成的六大方向进度
2. 每个真实功能闭环
3. API/CLI/MCP/Web/Docs/Test 变化
4. 本地验证结果
5. GitHub CI 结果
6. commit hash
7. push 结果
8. 仍未完成的生产 blocker
9. 当前成熟度判断

不要说 production-ready。最多说“更接近 enterprise production-candidate 的前置状态”。
