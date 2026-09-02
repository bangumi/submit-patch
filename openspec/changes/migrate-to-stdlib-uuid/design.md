## Context

项目使用 `github.com/gofrs/uuid/v5 v5.5.1`，用法集中在三类调用：`uuid.Must(uuid.NewV7())` / `uuid.Must(uuid.NewV4())`（生成主键、OAuth state、session key）和 `uuid.FromString(s)`（解析 URL/表单中的 patch id）。`uuid.UUID` 作为结构体字段出现在 HTTP JSON 响应（`routers.go`）和 sqlc 生成的 DAL 代码中，PostgreSQL 侧为原生 `uuid` 列，经 pgx v5 编解码。

Go 1.27 标准库新增 `uuid` 包（RFC 9562），本机 toolchain 已是 go1.27.0，`go.mod` 的 `go` 指令为 `1.26.7`。

## Goals / Non-Goals

**Goals:**

- 所有 `uuid` 引用切换到标准库 `uuid` 包，移除 gofrs 直接依赖
- `task build`、`task gen`、测试全部通过，行为不变

**Non-Goals:**

- 不改数据库 schema、不改 patch ID 的生成/解析语义
- 不处理其他间接依赖（如 `github.com/google/uuid`，由 `go mod tidy` 自行决定去留）

## Decisions

1. **API 映射保持语义一致**：
   - `uuid.Must(uuid.NewV7())` → `uuid.NewV7()`；`uuid.Must(uuid.NewV4())` → `uuid.NewV4()`。标准库的 `NewV4`/`NewV7` 不返回 error（随机源失败时 panic），`Must` 包装不再需要，调用点语义不变。
   - `uuid.FromString(s)` → `uuid.Parse(s)`。标准库 `Parse` 是 gofrs `FromString` 的超集（额外接受大括号、`urn:uuid:` 前缀形式），URL 中传入的规范形式两者都接受，不影响现有校验。
   - `uuid.UUID` 类型同名同布局（`[16]byte`），pgx v5 的 `UUIDCodec` 原生支持 `[16]byte`，DAL 无需改动逻辑。

2. **升级 `go` 指令到 `1.27.0`**：标准库 `uuid` 包要求 Go ≥ 1.27。替代方案（vendor 一份适配 1.26 的代码）不值得——项目 toolchain 已是 1.27，直接升级。

3. **更新 `sqlc.yaml` override 并重新 `task gen`**：`uuid` db_type 的 override 从 `github.com/gofrs/uuid/v5` 改为 `uuid`（标准库），重新生成 `dal/`，保证生成代码与手写代码使用同一个类型。

4. **保留现有错误处理**：`Parse` 失败路径沿用各 handler 现有的 `handleError` / 400 响应逻辑，只换函数名。

## Risks / Trade-offs

- [运行环境需要 Go ≥ 1.27] → CI 与部署镜像需同步升级；已在 proposal Impact 中注明。
- [`Parse` 接受的字符串形式变宽（大括号、urn 前缀）] → 仅影响显式提交的 patch id 解析，接受面变宽不构成安全或正确性问题；生成的 UUID 字符串表示两者一致。
- [sqlc 生成代码的 import 形式] → `task gen` 后检查 `dal/models.go`、`dal/query.sql.go` 的 import 与编译结果即可覆盖。
