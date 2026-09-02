# puzz 返回 HashWX Hashes

日期：2026-09-02
模块：`github.com/cxio/equix-cgo/puzz`
状态：已批准设计，待实现

在 `puzz` 子包为求解与校验补上与主包对等的 HashWX 哈希出口。puzz 每次只命中一个解，因此哈希作为多返回值，不引入 `Result`，也不改 `puzz.Solution`。

## 目标与非目标

**目标**

- 新增 `SolveWithHashes` / `SolveContextWithHashes` / `VerifyWithHashes`。
- 成功路径上，求解给出的 `equix.Hashes` 与校验给出的相同，且能通过 `equix.VerifyHashes`。
- 现有 `Solve` / `SolveContext` / `Verify` 的签名、语义、成本不变。

**非目标**

- 不给 `puzz.Solution` 增加 `Hashes` 字段，不改 24 字节编解码。
- 不引入 `puzz.Result`，不在 puzz 内重导出 `equix.Hashes`。
- 不把 `Verify` / `VerifyWithHashes` 改成返回 `error`（继续用 `bool`）。
- 不在 puzz 内包装 `equix.VerifyHashes`。
- 不新增 `Solver` / `Verifier` 类型；继续走主包池化入口。
- 不修改 `third_party/`、`internal/native/` 或主包公开 API。

## 公开 API

包名 `puzz`。类型 `equix.Hashes` 由调用方按需 `import "github.com/cxio/equix-cgo"`。

```go
func SolveWithHashes(challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error)

func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce uint64) (*Solution, equix.Hashes, error)

func VerifyWithHashes(challenge []byte, th Threshold, sol *Solution) (equix.Hashes, bool)
```

`SolveWithHashes` 是 `SolveContextWithHashes(context.Background(), …)` 的薄封装，与 `Solve` / `SolveContext` 的关系相同。

## 求解

搜索规则与现有 `SolveContext` 相同：进入时及每轮求解之间检查 `ctx`；每个 nonce 向主包求解一次；对每个候选做 SHA-256 阈值判定；未命中则 `nonce += 0x26f5`；当前轮已产出命中解时优先返回该结果，取消最多延迟一轮。

差异仅在主包入口：走 `equix.SolveWithHashesAndNonce`，命中时把该候选的 `Hashes` 一并返回。

```text
SolveContextWithHashes:
  loop:
    if ctx.Err() != nil → (nil, 零值 Hashes, ctx.Err())
    results, err := equix.SolveWithHashesAndNonce(challenge, nonce)
    if err != nil → (nil, 零值 Hashes, err)
    for r in results:
      if SHA256(challenge || nonce || r.Solution) 命中 th:
        return ({Nonce: nonce, Solution: r.Solution}, r.Hashes, nil)
    nonce += 0x26f5
```

错误或取消时解为 `nil`、哈希为零值。无解不是错误：搜索继续，直到命中、取消或主包报错。

不变量：同一 `(challenge, th, 起始 nonce)` 上，`SolveWithHashes` 命中的 `*Solution` 与 `Solve` 相同（主包保证同一 nonce 的解序列一致，puzz 按同一顺序取第一个过阈值的候选）。

现有 `Solve` / `SolveContext` **不得**改走 `SolveWithHashesAndNonce`。哈希提取有额外 `hashwx_exec` 成本，无哈希路径必须继续使用 `SolveWithNonce`。搜索循环允许小幅重复，或抽成私有辅助；辅助不得把无哈希路径变成带哈希路径。

## 校验

`VerifyWithHashes` 保持 puzz 的廉价过滤：先 SHA-256 阈值，通过后再做 Equi-X（此次同时取哈希）。

| 条件 | Hashes | bool |
| --- | --- | --- |
| `sol == nil` | 零值 | `false`（不 panic） |
| 阈值未过 | 零值 | `false`（不调用 Equi-X） |
| 阈值通过且 Equi-X 合法 | 该解的 8 个 HashWX | `true` |
| 阈值通过但 Equi-X 失败（`ErrOrder` / `ErrPartialSum` / `ErrFinalSum`） | 已算出的哈希 | `false` |
| 内部错误（分配失败等） | 零值 | `false` |

最后一行与现有 `Verify` 一致：puzz 的校验不暴露 `error`，内部失败也表现为 `false`。

阈值通过后调用 `equix.VerifyWithHashesAndNonce(challenge, sol.Nonce, sol.Solution)`。主包在结构校验失败时仍返回哈希；puzz 原样交出，bool 为 `false`。

不变量：对 `SolveWithHashes` 刚得到的 `(sol, h)`，`Verify(challenge, th, sol) == true`，`VerifyWithHashes` 返回 `(h, true)`，且 `equix.VerifyHashes(h) == nil`。`Hashes[i] = HashWX(seed, sol.Solution[i])`，seed 为 `challenge || little-endian(nonce)`。

## 文档

- `puzz/README.md`：补带哈希的用法示例（`SolveWithHashes` / `VerifyWithHashes`），并说明阈值未过时不计算 HashWX。
- `puzz/doc.go`：在包注释中链到上述三个函数。

不改根 `README.md`、不改 24 字节编码一节。

## 测试

在 `puzz/puzzle_test.go` 用 `go test ./puzz` 覆盖：

1. **往返**：`SolveWithHashes` 得到的解能被 `Verify` 接受；`VerifyWithHashes` 返回相同 `Hashes` 且 `true`；`equix.VerifyHashes(h) == nil`。
2. **与无哈希路径一致**：同一 `(challenge, th, 起始 nonce)` 上，`SolveWithHashes` 的 `*Solution` 等于 `Solve`。
3. **拒绝**：错误 nonce、篡改解、错误 challenge → `VerifyWithHashes` 为 `false`。这些情况通常先栽在阈值上，哈希为零值。
4. **nil**：`VerifyWithHashes(..., nil)` 为零值哈希 + `false`，不 panic。
5. **阈值通过、Equi-X 失败仍返回哈希**：用恒命中阈值 `FromBits(0)` 对合法解交换相邻索引，`VerifyWithHashes` 为 `false` 但哈希与 `equix.VerifyWithHashesAndNonce` 在同一 `(challenge, nonce, 打乱解)` 上给出的一致。
6. **取消**：已取消的 ctx 上 `SolveContextWithHashes` 返回 `context.Canceled`、`nil` 解、零值哈希。

不新增单独的成本基准；现有 `TestSolveCost` / `BenchmarkSolve` 仍走无哈希路径。

## 实现位置

仅改 `puzz/`：

- `puzz/puzzle.go`：三个导出函数
- `puzz/puzzle_test.go`：上列测试
- `puzz/README.md`、`puzz/doc.go`：文档
