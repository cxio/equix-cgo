# puzz 无 Nonce 一轮：Try / Accept

日期：2026-09-05
模块：`github.com/cxio/equix-cgo/puzz`
状态：已批准设计，待实现

在 `puzz` 为「调用方改 Challenge、包内不搜索」补上与现有 nonce 搜索对偶的一轮原语。难度仍是命中率（`Threshold`）。现有 `Solve` / `Verify` / `*WithHashes` / 24 字节 `Solution` 签名与语义不变。

## 目标与非目标

**目标**

- 在 `Threshold` 上新增 `Try` / `Accept`：对给定 Equi-X seed 做一轮求解或校验，并套用同一套 SHA-256 门槛。
- 统一哈希模型：门槛只看 `SHA256(seed || solution)`；现有 nonce 路径是 `seed = challenge || little-endian(nonce)` 的特例。
- 调用方通过改变 seed（即 Challenge 本身）搜索；本包不递增、不拼接 nonce。
- 现有公开 API 行为不变，包括 `Solve` 的无限搜索与 24 字节编码。

**非目标**

- 不改现有函数签名，不删 `Nonce`，不改 24 字节编解码。
- 不新增 `TryContext`、`TryWithHashes`、`AcceptWithHashes`。需要 HashWX 时调用方在命中后走 `equix.VerifyWithHashes(seed, sol)`。
- 不引入第二种 `Solution` 类型，不在 puzz 内重导出 `equix.Solution`。
- 不把 `Accept` 改成返回 `error`（继续用 `bool`）。
- 不新增 `Solver` / `Verifier`；继续走主包池化入口。
- 不修改 `third_party/`、`internal/native/` 或主包公开 API。
- 不把现有 `Solve` / `SolveContext` 改写成调用 `Try`（允许只抽共享哈希辅助）；`SolveWithHashes*` 仍走 `equix.SolveWithHashesAndNonce`。

## 公开 API

```go
func (th Threshold) Try(seed []byte) (*equix.Solution, error)
func (th Threshold) Accept(seed []byte, sol equix.Solution) bool
```

`seed` 就是 Equi-X 输入，原样交给 `equix.Solve` / `equix.Verify`。`nil` 与空切片合法，与主包一致。

`Threshold` 的构造（`FromProbability` / `FromBits`）与 `hit` 语义不变。

## 哈希

对任意 seed 与解：

```text
SHA256(seed || solution)[:8] -> 大端 uint64
该值 <= Threshold 即命中
```

`solution` 为 16 字节：`idx[0]…idx[7]` 各 2 字节 little-endian，与 `equix.Solution.MarshalBinary` 相同。

现有 `combinedHash(challenge, nonce, sol)` 必须与 `seedHash(challenge || le64(nonce), sol)` 字节级相同。nonce 仍用 little-endian，与主包 `appendNonce` 一致。阈值比较仍按大端解读哈希前 8 字节。

不变量：对任意合法 `puzz.Solution`，

```text
Verify(ch, th, sol) == th.Accept(ch || le64(sol.Nonce), sol.Solution)
```

`sol == nil` 时左边为 `false`，右边不适用（`Accept` 不接受指针）。

## 求解：Try

```text
Try(seed):
  sols, err := equix.Solve(seed)
  if err != nil → (nil, err)
  for sol in sols:          # 主包返回顺序
    if SHA256(seed || sol) 命中 th:
      return (sol 的副本指针, nil)
  return (nil, nil)
```

- 未命中（含 Equi-X 给出 0 个解，或有解但都未过门槛）是成功：`nil, nil`。不是错误，也不进入搜索循环。
- 只调用 `equix.Solve`，不得走 `SolveWithHashes*`。
- 同一 `seed` 上，`FromBits(0)` 若 Equi-X 有解，则 `Try` 返回的解等于 `equix.Solve(seed)[0]`。
- 不检查 `context`；一轮约数十毫秒。需要取消时由调用方在自己的 Challenge 循环里处理。

## 校验：Accept

与现有 `Verify` 相同的廉价过滤：先 SHA-256 门槛，通过后再 `equix.Verify`。

| 条件 | 结果 |
| --- | --- |
| 门槛未过 | `false`（不调用 Equi-X） |
| 门槛通过且 `equix.Verify(seed, sol) == nil` | `true` |
| 门槛通过但 Equi-X 失败（`ErrOrder` / `ErrPartialSum` / `ErrFinalSum` 或内部错误） | `false` |

`Accept` 按值接收 `equix.Solution`（`[8]uint16`），无 nil 指针问题。

不变量：对 `Try` 刚得到的非 nil `sol`，`th.Accept(seed, *sol) == true`。

## 内部实现

仅改 `puzz/`：

- `puzz/puzzle.go`：两个导出方法；将 `combinedHash` 抽成「先拼 seed、再哈希」的私有辅助，使 nonce 路径与 `Try`/`Accept` 共用同一条 SHA-256。现有 `Solve*` / `Verify*` 的控制流与主包入口不变。
- `puzz/puzzle_test.go`：下节用例。
- `puzz/README.md`、`puzz/doc.go`：补无 Nonce 用法；nonce 搜索仍是文档主路径。

不改根 `README.md`。

## 文档

`puzz/README.md` 增加一节「无 Nonce：调用方改 Challenge」：

- 说明门槛公式的一般形式是 `SHA256(seed || solution)`，nonce 路径是 seed 拼接特例。
- 示例：`th.Try(challenge)`，未命中则换 challenge 再试；`th.Accept` 校验。
- 写明：需要 HashWX 时在命中后调用 `equix.VerifyWithHashes`，本包不提供 `TryWithHashes`。

`puzz/doc.go` 在包注释中链到 `Try` / `Accept`，并写明未命中返回 `nil, nil`。

## 测试

在 `puzz/puzzle_test.go` 用 `go test ./puzz` 覆盖：

1. **命中往返**：找到一个 Equi-X 有解的 seed（可对固定前缀追加计数直到 `equix.Solve` 非空），`FromBits(0)` 下 `Try` 返回非 nil，且等于 `equix.Solve(seed)[0]`；`Accept` 为 `true`。
2. **拒绝**：篡改解、错误 seed → `Accept` 为 `false`。
3. **未命中**：`Threshold(0)`（前 64 位全零才命中，实际几乎不可达）对上述有解 seed 返回 `nil, nil`；同一解的 `Accept` 为 `false`（除非该解哈希恰好为 0，此时换 seed 再断言）。
4. **与 nonce 路径一致**：现有 `Solve` 得到的 `sol`，`th.Accept(appendLE64(challenge, sol.Nonce), sol.Solution)` 为 `true`，且与 `Verify(challenge, th, sol)` 相同。测试内可自写 8 字节 little-endian 拼接，不调用主包未导出符号。
5. **空 seed**：`Try(nil)` 与 `Try([]byte{})` 不 panic；要么 `nil, nil`，要么返回能被 `Accept` 接受的解。
6. **现有测试全过**：`Solve` / `Verify` / `*WithHashes` / 编解码 / 成本测试行为不变。

不新增单独的成本基准；`TestSolveCost` / `BenchmarkSolve` 仍走 nonce 搜索路径。

## 调用示例

```go
th, err := puzz.FromBits(puzz.DefaultBits)
if err != nil {
    panic(err)
}

sol, err := th.Try(challenge)
if err != nil {
    panic(err)
}
if sol == nil {
    // 本轮未命中：调用方改 challenge 再 Try
    return
}

if !th.Accept(challenge, *sol) {
    panic("unexpected reject")
}

// 需要 HashWX 时：
h, err := equix.VerifyWithHashes(challenge, *sol)
_ = h
```
