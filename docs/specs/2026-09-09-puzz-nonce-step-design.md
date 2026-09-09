# puzz nonce 搜索步进参数

日期：2026-09-09
模块：`github.com/cxio/equix-cgo/puzz`
状态：已批准设计，待实现

为 `puzz` 带 nonce 的搜索入口增加步进参数。调用方可指定正值步进；`0` 表示沿用现有默认 `nonceStep`（`0x26f5`）。这是破坏性改签名。`Try` / `Accept` / `Verify*` 不搜索 nonce，签名与语义不变。

## 目标与非目标

**目标**

- 四个 nonce 搜索入口统一增加最后参数 `step uint64`：`Solve`、`SolveContext`、`SolveWithHashes`、`SolveContextWithHashes`。
- `step == 0` 解析为包内默认 `nonceStep`；非 0 值原样作为步进。解析在进入搜索循环前做一次。
- `Solve*(challenge, th, nonce, 0)` 与改签名前的行为相同（同一条算术序列 `nonce, nonce+0x26f5, …`）。
- 文档示例改为末参传 `0`；多 worker 说明改为「同一步进下起点避免相差步进的整数倍」。

**非目标**

- 不导出 `nonceStep` / `NonceStep`（同包测试可直接用未导出常量）。
- 不新增 `SolveStep` 一类函数，不保留旧三参数签名。
- 不改 `Try` / `Accept` / `Verify` / `VerifyWithHashes`。
- 不改 `puzz.Solution`、24 字节编解码、哈希公式、取消语义。
- 不为非法步进返回 `error`：`uint64` 无负值，`0` 是唯一哨兵。
- 不修改 `third_party/`、`internal/native/` 或主包公开 API。
- 不把无哈希路径改走 `SolveWithHashesAndNonce`。

## 公开 API

```go
func Solve(challenge []byte, th Threshold, nonce, step uint64) (*Solution, error)

func SolveContext(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, error)

func SolveWithHashes(challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error)

func SolveContextWithHashes(ctx context.Context, challenge []byte, th Threshold, nonce, step uint64) (*Solution, equix.Hashes, error)
```

`Solve` / `SolveWithHashes` 仍分别是对应 `*Context*` 在 `context.Background()` 上的薄封装，并把 `step` 原样下传。

有效步进：

```text
effectiveStep(step) = nonceStep  if step == 0
                    = step       otherwise
```

循环内使用已解析的有效步进，不得每轮再把 `0` 当成「再解析一次」。`nonce += effectiveStep` 按 `uint64` 自然溢出，与现有加法行为相同。

## 求解

搜索规则与现有一致，仅把写死的 `nonce += 0x26f5` 换成解析后的步进：

```text
SolveContext:
  step = effectiveStep(step)
  loop:
    if ctx.Err() != nil → (nil, ctx.Err())
    sols, err := equix.SolveWithNonce(challenge, nonce)
    if err != nil → (nil, err)
    for sol in sols:          # 主包返回顺序
      if SHA256(challenge || nonce || sol) 命中 th:
        return ({Nonce: nonce, Solution: sol}, nil)
    nonce += step
```

`SolveContextWithHashes` 同样先解析 `step`，主包入口仍是 `equix.SolveWithHashesAndNonce`；未命中则 `nonce += step`。取消、错误、无解继续搜的语义不变。

不变量：

1. **默认等价**：同一 `(challenge, th, 起始 nonce)` 上，`Solve(…, 0)` 与 `Solve(…, nonceStep)` 的 `*Solution` 相同（`SolveWithHashes` 对自身同样成立）。
2. **取模**：命中 nonce 满足 `(sol.Nonce - start) % effectiveStep == 0`（`uint64` 减法与取模）。
3. **两路径一致**：同一 `(challenge, th, 起始 nonce, step)` 上，`SolveWithHashes` 命中的 `*Solution` 与 `Solve` 相同。
4. **校验无关步进**：`Verify` / `VerifyWithHashes` 只看提交里的 nonce，不接收、不依赖搜索步进。

多 worker：各用不同起点，或不同且与 `2^64` 互素的步进。使用相同有效步进时，起点仍应避免相差该步进的整数倍，否则序列完全重合。默认步进仍是奇素数 `0x26f5`，避免与 2 的幂步进对齐。

## 文档

- `puzz/README.md`：所有 `Solve*` 示例补上末参 `0`；「难度预算」一节写明 `step==0` 用默认 `0x26f5`，自定义正值覆盖；多 worker 重叠条件改为相对有效步进。
- `puzz/doc.go`：包注释中「以素数 0x26f5 步进」改为「默认以素数 0x26f5 步进，可由 `step` 覆盖（0 表示默认）」。

不改根 `README.md`。

## 测试

在 `puzz/puzzle_test.go` 用 `go test ./puzz` 覆盖：

1. **现有用例补 `0`**：所有对四个搜索函数的调用补上末参 `0`，行为与改签名前相同。`TestSolveCost` 的轮数计算仍用 `nonceStep`（传入 `0` 即默认步进）。
2. **默认等价**：同一 challenge / 阈值 / 起点上，`Solve(…, 0)` 与 `Solve(…, nonceStep)` 的解相同；`SolveWithHashes` 对 `0` 与 `nonceStep` 同样成立。
3. **取模**：对步进 `1` 与 `7`（均非 0），`Solve` 命中 nonce 满足 `(sol.Nonce-start)%step == 0`，且 `Verify` 接受该解。用 `FromBits(DefaultBits)` 或 `FromProbability(0.1)` 即可，不必构造「第一轮必 miss」。
4. **两路径一致（自定义步进）**：同一 `(challenge, th, start, 7)` 上，`SolveWithHashes` 的 `*Solution` 等于 `Solve`。
5. **取消仍成立**：已取消的 ctx 上 `SolveContext` / `SolveContextWithHashes` 在传入非 0 `step` 时仍立即返回 `context.Canceled`（步进尚未使用）。

不新增单独的成本基准；`TestSolveCost` / `BenchmarkSolve` 继续传 `step=0`。

## 实现位置

仅改 `puzz/`：

- `puzz/puzzle.go`：四个导出函数签名；一处私有 `effectiveStep`（或等价内联）；两处搜索循环使用解析后的步进。
- `puzz/puzzle_test.go`：现有调用补 `0`；上列新用例。
- `puzz/README.md`、`puzz/doc.go`：文档。
