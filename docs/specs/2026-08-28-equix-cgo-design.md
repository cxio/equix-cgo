# Equi-X Cgo 封装

日期：2026-08-28
模块：`github.com/cxio/equix`
状态：已批准设计，待实现

把官方 Equi-X C 实现（`tevador/equix` 的 `equix_v2` 分支）封装成可被其它 Go 项目 `go get` 共享的包。算法走 Equi-X v2（HashWX），不暴露 v1。

## 目标与非目标

**目标**

- 提供 `Solve` / `Verify`，以及可复用的 `Solver` / `Verifier`。
- 另提供带 `uint64` nonce 的 API：nonce 由调用方固定，challenge 由外部变化；本包不在内部搜索或递增 nonce。
- 另提供返回解与 8 个 HashWX 哈希值的 API（`SolveWithHashes` / `SolveWithHashesAndNonce`）。
- `go get github.com/cxio/equix` 即可使用（调用方需有 C 编译器）。
- 与官方 C 在 v2 路径上行为一致。

**非目标**

- 不暴露 Equi-X v1、HashX/HashWX 直接 API、`COMPILE`、大页。
- 不提供 `CGO_ENABLED=0` 的纯 Go 实现。
- 不要求调用方安装 CMake 或系统 `libequix`。
- 不修改 `third_party/` 里的上游 C 源码。

## 上游钉死版本

从 GitHub 以源码快照形式放入仓库（不用 git submodule：Go module zip 不含 submodule）。

| 仓库 | 引用 | Commit |
|------|------|--------|
| [tevador/equix](https://github.com/tevador/equix) | 分支 `equix_v2` | `350a85dedda1344637dac09a1de786ee63a5fb01` |
| [tevador/hashx](https://github.com/tevador/hashx) | 该分支 submodule | `08babdf4f41b0b8991d1fa94914c7c6902de0cb6` |
| [tevador/hashwx](https://github.com/tevador/hashwx) | 该分支 submodule | `d771cbf6cdc070755f7d137cdcf9d781af14da3f` |

升级时覆盖 `third_party/` 对应目录，更新本表与 `internal/native` 中的版本戳（见构建），不改 C 文件内容。

## 目录结构

```
github.com/cxio/equix
  equix.go              # 包级 Solve / Verify / *WithNonce / *WithHashes
  solution.go           # Solution、Hashes、Result 与编解码
  solver.go             # Solver
  verifier.go           # Verifier
  errors.go             # 导出错误值
  equix_test.go
  internal/native/      # 唯一 import "C" 的包：cgo 指令、stub .c、取哈希胶水、版本戳
  third_party/equix/    # 上游快照（含 LICENSE）
  third_party/hashx/
  third_party/hashwx/
```

公开包 `equix` 不直接 `import "C"`，只调用 `internal/native`。这样公开 API 与 cgo 边界分开。

## 公开 API

包名 `equix`。

### Solution

```go
type Solution [8]uint16
```

对应 C 的 `equix_solution.idx[8]`。一次求解最多 `EQUIX_MAX_SOLS`（8）个解。

- `MarshalBinary() ([]byte, error)`：16 字节，每个索引为 little-endian `uint16`，顺序 `idx[0]…idx[7]`。
- `UnmarshalBinary([]byte) error`：恰好 16 字节，按上述规则解码；长度不对则报错。

不提供十六进制辅助函数。

### Hashes 与 Result

每个解对应 8 个 HashWX 输出，与 `Solution` 下标一一对应：`Hashes[i] = HashWX(seed, Solution[i])`，类型为 `uint64`（HashWX 的完整 64 位结果，不是截断到 60 位）。

```go
type Hashes [8]uint64

type Result struct {
    Solution Solution
    Hashes   Hashes
}
```

不给 `Hashes` 做二进制编解码。合法解上 `Hashes[0]+…+Hashes[7] ≡ 0 (mod 2^60)`（与 Equi-X 定义一致）；本包不在公开 API 里再验一次，由求解器与测试保证。

哈希在 **同一次** `equix_solve` 之后、同一 `equix_ctx` 上立刻用 `hashwx_exec` 取出（challenge 已在 solve 里做成 HashWX 实例）。胶水 C 放在 `internal/native`（本仓库代码），读取官方内部 `context.h`，**不修改** `third_party/`。

### 无 nonce

Equi-X 输入即 `challenge`（可为 `nil` 或空切片，长度为 0）。

```go
func Solve(challenge []byte) ([]Solution, error)
func Verify(challenge []byte, sol Solution) error

func NewSolver() (*Solver, error)
func (s *Solver) Solve(challenge []byte) ([]Solution, error)
func (s *Solver) Close() error

func NewVerifier() (*Verifier, error)
func (v *Verifier) Verify(challenge []byte, sol Solution) error
func (v *Verifier) Close() error
```

`Solve` 在 0 个解时返回空切片和 `nil` error（部分 challenge 本来无解）。切片长度范围为 0–8。若 C 返回的解个数不在 0–8，视为内部错误（不是上述哨兵）。

### 带 nonce

Equi-X 输入为 **`challenge` 后接 8 字节 little-endian `uint64` nonce**。

```go
func SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error)
func VerifyWithNonce(challenge []byte, nonce uint64, sol Solution) error

func (s *Solver) SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error)
func (v *Verifier) VerifyWithNonce(challenge []byte, nonce uint64, sol Solution) error
```

不变量：`SolveWithNonce(ch, n)` 与 `Solve(append(ch, le64(n)...))` 的解集一致；`VerifyWithNonce` 与对应的 `Verify` 一致。`nonce == 0` 合法。本包不递增 nonce；调用方通过改变 `challenge`、固定 `nonce` 来搜索。

无 nonce 的 `Solve`/`Verify` **不**附加这 8 字节。

### 带哈希

在对应的 `Solve` / `SolveWithNonce` 之上，额外返回每个解的 `Hashes`。索引解集与不带哈希的 API 相同。

```go
func SolveWithHashes(challenge []byte) ([]Result, error)
func SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error)

func (s *Solver) SolveWithHashes(challenge []byte) ([]Result, error)
func (s *Solver) SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error)
```

不变量：`SolveWithHashes(ch)` 的 `.Solution` 序列与 `Solve(ch)` 相同；`SolveWithHashesAndNonce(ch, n)` 相对 `SolveWithNonce(ch, n)` 同样成立。`Verify` / `VerifyWithNonce` 仍只吃 `Solution`，不接收 `Hashes`。

### 不导出

HashX、HashWX、`EQUIX_V2`、`COMPILE`、大页、v1、原始 `equix_ctx` 指针均不导出。

## Context 分配与 JIT 回退

分配时始终带 `EQUIX_V2`。

- `NewSolver`：`EQUIX_CTX_SOLVE | EQUIX_V2 | EQUIX_CTX_COMPILE`；若返回 `EQUIX_NOTSUPP`，再试 `EQUIX_CTX_SOLVE | EQUIX_V2`（解释器）。
- `NewVerifier`：`EQUIX_CTX_VERIFY | EQUIX_V2 | EQUIX_CTX_COMPILE`，同样在 `EQUIX_NOTSUPP` 时去掉 `COMPILE`。

两次都失败：`equix_alloc` 返回 `NULL` 视为内存错误；两次都是 `EQUIX_NOTSUPP` 则 `ErrNotSupported`。

不使用 `EQUIX_CTX_HUGEPAGES`。

HashX 仍会编译进二进制（官方 `context.c`/`equix.c` 含 v1 分支），Go 路径永不设置 v1、不导出 v1。

## 并发与生命周期

C 的 `equix_ctx` 非线程安全。

- 同一个 `Solver` 或 `Verifier` 不得被多个 goroutine 同时使用。需要并行时各用各的实例。
- 包级函数使用两个 `sync.Pool`：solver 池（~1.8 MiB/个）与 verifier 池。每次调用 `Get`，用完 `Put`。池中对象在进程内不 `equix_free`。
- 池的 `New` 与 `NewSolver`/`NewVerifier` 使用同一套 JIT 回退。池分配失败时，该次包级调用返回错误（不缓存失败对象）。
- `Close` 幂等；对 `nil` receiver 安全（不 panic）。Close 后再调用 `Solve`/`Verify` 及其带 nonce、带哈希的变体，返回 `ErrClosed`。
- 显式实例由调用方 `Close`；未 Close 会泄漏对应 C 堆（solver 约 1.8 MiB）。`runtime.SetFinalizer` 不作要求。

## 错误

可 `errors.Is` 的值：

| 值 | 含义 |
|----|------|
| `ErrNotSupported` | 编译型与解释型 context 都无法分配 |
| `ErrChallenge` | C `EQUIX_CHALLENGE`（v2 上几乎不会出现；若上游返回则映射） |
| `ErrOrder` | C `EQUIX_ORDER` |
| `ErrPartialSum` | C `EQUIX_PARTIAL_SUM` |
| `ErrFinalSum` | C `EQUIX_FINAL_SUM` |
| `ErrClosed` | 使用已 Close 的 `Solver`/`Verifier` |

`Verify` 成功返回 `nil`。其它失败（含 `UnmarshalBinary` 长度错误、C 分配 `NULL`）用普通 `error`，不必是上述哨兵。

## Cgo 构建

`internal/native` 是唯一含 `import "C"` 的目录。cgo 只编译该目录下的 `.c`，因此每个上游翻译单元对应一个薄 stub，内容仅为 `#include` 那一份上游 `.c`。

**宏（与官方 CMake 对齐）**

- `HASHX_SIZE=8`
- `HASHX_STATIC`、`HASHWX_STATIC`、`EQUIX_STATIC`

**编译选项**

- `-std=c11`（HashWX 需要 C11）
- `-O2`（可移植；不加 `-march=native`）
- Include：`third_party/equix/include`、`third_party/equix/src`（胶水读取 `context.h`）、`third_party/hashx/include`、`third_party/hashx/src`、`third_party/hashwx/include`，以及 hashwx 的 `src`（若编译需要）

**链接**

- Windows：`#cgo windows LDFLAGS: -ladvapi32`（hashx `virtual_memory.c`）
- 其它平台无额外库。不链接 pthread（官方仅 bench 使用线程）。

**编译的上游 .c（不含 tests/bench）**

- equix：`context.c`、`equix.c`、`solver.c`
- hashx：`blake2.c`、`compiler.c`、`compiler_a64.c`、`compiler_x86.c`、`context.c`、`hashx.c`、`program.c`、`program_exec.c`、`siphash.c`、`siphash_rng.c`、`virtual_memory.c`
- hashwx：`compiler.c`、`compiler_a64.c`、`compiler_wasm.c`、`compiler_x86.c`、`context.c`、`hashwx.c`、`program.c`、`program_exec.c`、`siphash_rng.c`、`virtual_memory.c`

**本仓库胶水（非上游）**

- `internal/native` 中一份 `.c`：在 `equix_solve` 之后对每个解调用 8 次 `hashwx_exec`，写入 `uint64[8]`。只用于 Equi-X v2 context。

**版本戳**

`internal/native` 内导出或未导出的字符串常量，写入上述三个 commit SHA。升级 `third_party/` 时必须改此文件，以便 cgo 在上游子目录变更时也会重编（Go 默认不跟踪其它目录的 `.c` 变更）。

**平台**

Linux / macOS / Windows × amd64 / arm64。Windows 使用 MinGW（或兼容的 gcc）作为 cgo 工具链，不支持 MSVC。`CGO_ENABLED=0` 时本模块不能构建。

**Go 版本**

与现有 `go.mod` 一致：`go 1.27.0`。

## 许可

- 本仓库 Go 代码：MIT（根目录 `LICENSE`）。
- `third_party/equix`、`hashx`、`hashwx`：LGPL-3.0，保留各目录原 `LICENSE`。
- README 必须说明：静态链入官方 C 构成 LGPL 组合作品；分发二进制时需保留 LGPL 声明并提供对应 C 源码（本仓库已包含）。

## 测试

用 `go test`。

1. **官方 `tests.c` 的 v2 路径**：`EQUIX_CTX_SOLVE | EQUIX_V2` 下对递增的 4 字节 nonce 求解（与官方测试相同的搜索方式仅用于造解，不是公开 API）；`Verify` 成功；交换相邻索引 → `ErrOrder`；按官方 `test_verify3`/`test_verify4` 的交换 → `ErrOrder` / `ErrPartialSum`；8 个索引的 40320 种排列中恰好一种 `Verify` 为 `nil`。
2. **nonce API**：同一 `(challenge, nonce)` 上 `SolveWithNonce` 的解能被 `VerifyWithNonce` 接受；只改 nonce 或只改 challenge 则失败。
3. **拼接不变量**：若 `SolveWithNonce(ch, n)` 得到解 `s`，则 `Verify(append(ch, le64(n)...), s)` 成功，且 `Solve(append(ch, le64(n)...))` 的解集与前者一致（在同一 challenge 上）。
4. **带哈希**：`SolveWithHashes` 与 `Solve` 的 `Solution` 序列一致；每个 `Result` 的 `Hashes[i]` 与 `Solution[i]` 对齐，且八个 `uint64` 之和的低 60 位为 0。`SolveWithHashesAndNonce` 对 `SolveWithNonce` 做同样断言。
5. **边界**：空 challenge；合法的 0 解（不报错）；`Close` 后再用 → `ErrClosed`；`Close` 两次不 panic。
6. 可选 `BenchmarkSolve` / `BenchmarkVerify`，不作为 CI 失败条件。

## 调用示例

```go
sols, err := equix.SolveWithNonce(challenge, nonce)
if err != nil { ... }
for _, sol := range sols {
    if err := equix.VerifyWithNonce(challenge, nonce, sol); err != nil { ... }
}

solver, err := equix.NewSolver()
if err != nil { ... }
defer solver.Close()
sols, err = solver.SolveWithNonce(challenge, nonce)

results, err := equix.SolveWithHashesAndNonce(challenge, nonce)
for _, r := range results {
    _ = r.Solution
    _ = r.Hashes // [8]uint64，与 Solution 下标对齐
}
```

## 实现顺序

1. 写入 `third_party/` 快照与 stub、cgo 编译通过。
2. `internal/native` 最小 C 绑定（alloc/free/solve/verify）。
3. 公开类型、错误、无 nonce API。
4. `SolveWithNonce` / `VerifyWithNonce`。
5. `SolveWithHashes` / `SolveWithHashesAndNonce`（solve 后同 ctx 取 8 个哈希）。
6. `sync.Pool` 包级函数。
7. 测试与 README（含许可与上游 commit）。
