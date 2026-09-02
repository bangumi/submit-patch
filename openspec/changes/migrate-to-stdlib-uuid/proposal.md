## Why

Go 1.27 在标准库中新增了 `uuid` 包（RFC 9562），项目当前依赖 `github.com/gofrs/uuid/v5 v5.5.1`。迁移到标准库可以移除一个直接依赖，且新代码默认使用官方维护的实现。

## What Changes

- 将所有代码中的 `github.com/gofrs/uuid/v5` 导入替换为标准库 `uuid` 包
- API 映射：
  - `uuid.Must(uuid.NewV7())` → `uuid.NewV7()`（标准库 `NewV7` 直接返回 `UUID`，无 error）
  - `uuid.Must(uuid.NewV4())` → `uuid.NewV4()`（同上）
  - `uuid.FromString(s)` → `uuid.Parse(s)`
- `go.mod` 的 `go` 指令升级到 `1.27.0`（标准库 `uuid` 包要求）
- `sqlc.yaml` 中 `uuid` 类型的 override 从 gofrs 包改为标准库 `uuid` 包，并重新运行 `task gen`
- 通过 `go mod tidy` 移除 `github.com/gofrs/uuid/v5` 依赖

外部行为不变：UUID 格式（v4/v7、小写连字符表示）、JSON 序列化（`MarshalText`）、pgx 的 `[16]byte` 编解码均与现有一致。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

（无 — 本变更为纯内部重构，不改任何 spec 级别行为，已在 `.openspec.yaml` 声明 `skip_specs: true`）

## Impact

- 受影响文件：`episode.go`、`subject.go`、`character.go`、`person.go`、`auth.go`、`review.go`、`*-review.go`、`routers.go`、`sqlc.yaml`、`go.mod`，以及 `task gen` 重新生成的 `dal/` 代码
- 依赖：移除直接依赖 `github.com/gofrs/uuid/v5`；`github.com/google/uuid` 为间接依赖，是否保留由 `go mod tidy` 决定
- 数据库 schema 不变（PostgreSQL `uuid` 列类型不变）
- 运行环境要求 Go ≥ 1.27
