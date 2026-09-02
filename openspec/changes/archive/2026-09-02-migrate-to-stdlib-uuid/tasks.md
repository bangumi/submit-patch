## 1. 代码与配置迁移

- [x] 1.1 将 `go.mod` 的 `go` 指令从 `1.26.7` 升级到 `1.27.0`，确认 `go build ./...` 在新指令下通过
- [x] 1.2 将所有 `.go` 文件的 `github.com/gofrs/uuid/v5` 导入替换为标准库 `uuid` 包，并按映射改写调用点：`uuid.Must(uuid.NewV7())` → `uuid.NewV7()`、`uuid.Must(uuid.NewV4())` → `uuid.NewV4()`、`uuid.FromString` → `uuid.Parse`；验证 `gofmt -l .` 无输出
- [x] 1.3 更新 `sqlc.yaml` 中 `uuid` 类型的 override 为标准库 `uuid` 包，运行 `task gen` 重新生成 `dal/`，检查生成代码的 import 与编译通过

## 2. 依赖清理与验证

- [x] 2.1 运行 `go mod tidy`，确认 `go.mod` 不再包含 `github.com/gofrs/uuid/v5` 直接依赖，`git diff go.mod go.sum` 无意外变更
- [x] 2.2 运行 `go build ./...` 和 `go vet ./...`，确认全部通过
- [x] 2.3 运行 `go test ./...`，确认全部测试通过
- [x] 2.4 运行 `task build`，确认二进制构建成功
