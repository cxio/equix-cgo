# AGENTS.md

Go cgo 封装，绑定 tevador/equix 的 `equix_v2` 分支（HashWX）。单 module `github.com/cxio/equix-cgo`，无第三方 Go 依赖，无 CI/lint 配置（静态检查靠 `go vet`）。

## 命令

需要 `CGO_ENABLED=1` 和 C 编译器（gcc/clang；Windows 用 MinGW）。不支持 `CGO_ENABLED=0`；交叉编译需对应目标平台的 C 工具链。

```bash
go vet ./... && go test ./...                  # 全量验证（含计时类测试，约 10s）
go test -run TestSolveVerify .                 # 单个测试
go test ./puzz -run TestFromBits               # 单个子包测试
go test -run 'TestSolveCost' ./puzz -v -count=1   # 计时类测试要 -v 才输出统计
go test -bench 'Benchmark(Solve|Verify)$' -benchmem   # 主包基准
go test ./puzz -bench .
```

## 架构

- `internal/native` 是**唯一** `import "C"` 的包；公开包 `equix` 与 `puzz` 不直接碰 cgo。API 与 cgo 边界分离是设计约束（见 `docs/specs/`）。
- cgo 只编译包目录内的 `.c`，因此每个上游翻译单元在 `internal/native/` 有一个一行 stub（`stub_<lib>_<file>.c`），内容仅为 `#include "../../third_party/..."`。这些不是重复代码，别删；上游新增 `.c` 时需要补 stub。
- `third_party/{equix,hashx,hashwx}` 是钉死 commit 的上游源码快照（刻意不用 git submodule——Go module zip 不含 submodule）。**禁止修改**其中任何文件。
- 升级上游要同步三处：`third_party/` 源码、`README.md` 的 commit 表、`internal/native/version.go`（`TestVersionPins` 会校验）。
- `puzz/` 是在 equix 之上叠加 SHA-256 阈值的 PoW 封装，详见 `puzz/README.md`。
- `docs/specs/`、`docs/plans/` 是设计与实施计划，是 API 契约的权威来源；与代码冲突时以实现为准，但改动 API 前先读它。

## 约定

- 注释、文档、commit message 用简体中文；错误消息（`errors.New`、`fmt.Errorf` 等）用英文。
- C 编译选项集中在 `internal/native/cgo.go`：`-std=c11`（HashWX 需要）、`-O2`（刻意不加 `-march=native`，保证可移植）、`HASHX_SIZE=8` 与 `*_STATIC` 宏对齐上游 CMake。Windows 额外链 `-ladvapi32`。
- 许可：Go 代码 MIT，`third_party/` C 代码 LGPL-3.0；静态链入后分发二进制须附带这些 C 源码。

## 易错点

- `VerifyWithHashes*` 路径必须**先** `hashwx_make` 再 `equix_verify`：官方 C 在 `EQUIX_ORDER` 失败时不会生成 HashWX，顺序反了会读到未初始化内存。
- `Solve` 无解时返回**空切片 + `nil` error**（部分 challenge 本来无解），既不是 `nil` 切片也不是错误。
- `Solver` / `Verifier` 实例非线程安全（同一实例串行用，Close 幂等）；包级函数经 `sync.Pool` 可并发。C 堆靠 `runtime.AddCleanup` 兜底回收，调用方仍应显式 `Close`。
