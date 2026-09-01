# puzz

在 Equi-X 求解之上叠加一层可调难度的 SHA-256 门槛，提供"客户端搜索、服务端验证"的 PoW 谜题封装。本包由原 `ratio`（按概率调难度）与 `xbits`（按前缀零位数调难度）两个包合并而来——两者本是同一机制的两种参数化，现在统一为单一的 `Threshold` 阈值模型。

## 机制

对每个候选解计算：

```text
SHA256(challenge || nonce || solution)[:8] -> 大端 uint64
```

该数值 `<= Threshold` 即命中。两种构造方式对应同一机制：

```go
// 按期望命中概率：约 10% 的候选解命中
th, err := puzz.FromProbability(0.1)

// 按前缀零位数：前 4 位全零，等价于 6.25% 命中概率
th, err := puzz.FromBits(puzz.DefaultBits)
```

命中概率的精确值为 `(th+1)/2^64`，与参数的偏差不超过 `2^-64`。非法参数（`p <= 0`、`p > 1`、`NaN`、`bits` 越界）一律返回错误——不会静默产出不可达的阈值让求解无限空转。

## 用法

```go
package main

import (
	"fmt"

	"github.com/cxio/equix-cgo/puzz"
)

func main() {
	challenge := []byte("equix-cgo/puzzle example")

	th, err := puzz.FromBits(puzz.DefaultBits)
	if err != nil {
		panic(err)
	}

	// 客户端：从 nonce=13 起搜索
	sol, err := puzz.Solve(challenge, th, 13)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found solution! Nonce: %d\n", sol.Nonce)

	// 服务端：先哈希门槛过滤，再完整 Equi-X 校验
	fmt.Printf("Verification result: %v\n", puzz.Verify(challenge, th, sol))
}
```

## 取消与限时

`Solve` 的搜索没有天然终点，可能永不返回。需要限时或取消时使用 `SolveContext`：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

sol, err := puzz.SolveContext(ctx, challenge, th, 13)
if errors.Is(err, context.DeadlineExceeded) {
    // 超时未命中
}
```

取消在每个求解轮（约数十毫秒）之间生效；若当前轮已产出命中解，仍优先返回该有效结果。

## 难度预算的口径

命中期望需要约 `1/p`（或 `2^bits`）个 **Equi-X 候选解**。Equi-X 每个 nonce 平均产出约 1.7 个候选解，因此折合约 `0.6/p`（或 `2^bits/1.7`）**轮 nonce 尝试**。例如 `DefaultBits=4`：期望约 16 个候选解，对应约 9 轮 nonce。评估成本时注意区分这两个单位。

nonce 步进为素数 `0x26f5`（9973），避免多 worker 的搜索序列与 2 的幂步进对齐；多 worker 并行分片时，各起点应避免相差 `0x26f5` 的整数倍，否则搜索序列完全重叠。

## 验证成本

`Verify` 先做阈值比对（一次 SHA-256，约百纳秒），通过后再做完整的 Equi-X 结构校验（约 10µs 量级）。无效提交的平均验证成本接近一次哈希，适合服务端直接面对不可信输入。`sol` 为 `nil` 时返回 `false`，不会 panic。

## 实测

Mac mini M4 Pro（`DefaultBits=4`），来自 `go test ./puzz -run TestSolveCost -v -count=1`：

```text
solve  min=57.972625ms avg=245.10727ms max=660.531042ms
verify min=17µs avg=39.114µs max=76.916µs
tries  min=2 avg=8.5 max=23 (DefaultBits=4, expected≈9.4 nonce rounds)
```

实测平均 8.5 轮 nonce 与理论预期（约 9.4 轮）吻合。

## 并发

所有函数内部经主包的池化入口工作，可从多个 goroutine 并发调用；`Solution` 自身不含共享状态。需要压榨服务端验证吞吐时，可绕过本包、自行持有 `equix.Verifier` 实例并复用 `Threshold` 比较（实例复用约 6µs/次，包级入口约 20-40µs/次）。

## 解的 24 字节编码

`Solution` 实现 `MarshalBinary` / `UnmarshalBinary`：8 字节 little-endian `Nonce` 后接 16 字节解（`idx[0]`…`idx[7]`，各 2 字节 little-endian），恰好 24 字节。

```go
b, err := sol.MarshalBinary()
if err != nil {
    return err
}
var sol2 puzz.Solution
if err := sol2.UnmarshalBinary(b); err != nil {
    return err
}
```

## 从旧包迁移

| 旧 API | 新 API |
| --- | --- |
| `ratio.TargetFromProbability(p)` | `puzz.FromProbability(p)`（返回 error） |
| `ratio.Target` | `puzz.Threshold` |
| `ratio.Solve / Verify` | `puzz.Solve / Verify`（签名相同） |
| `xbits.TargetBits` | `puzz.DefaultBits` |
| `xbits.Solve / Verify` | `puzz.Solve / Verify` 配合 `puzz.FromBits(...)` |

## 许可

与主仓库一致：Go 代码为 MIT，静态链入的 Equi-X/HashX/HashWX C 代码为 LGPL-3.0。
