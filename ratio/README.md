# ratio

基于概率阈值的哈希前缀难度封装。它不是固定位数门槛，而是把“目标成功概率”映射成一个 `uint64` 阈值：

```go
threshold := ratio.TargetFromProbability(0.1) // 10% 目标概率
```

随后在每个候选 Equi-X 解上，计算：

```text
SHA256(challenge || nonce || solution)[:8] -> uint64
```

若该数值 `<= threshold`，则视为命中目标；否则继续尝试下一 nonce。

## 设计思路

`TargetFromProbability` 直接将期望概率转换成 `uint64` 上限：

```go
return uint64(float64(math.MaxUint64) * p)
```

因此：

- `p <= 0` -> `0`
- `p >= 1` -> `math.MaxUint64`
- `p = 0.1` -> 约为 `10%` 的 `uint64` 阈值

这让求解过程可以通过概率参数快速调节难度，而不需要再手工维护位数门槛。

## 用法

```go
package main

import (
    "fmt"

    "github.com/cxio/equix-cgo/ratio"
)

func main() {
    challenge := []byte("equix-cgo/ratiox cost")
    target := ratio.TargetFromProbability(0.1)

    // 用一个小素数作为 nonce 起点值
    sol, err := ratio.Solve(challenge, target, 13)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found solution at nonce=%d\n", sol.Nonce)

    ok := ratio.Verify(challenge, target, sol)
    fmt.Printf("Verification result: %v\n", ok)
}
```

## 内部流程

求解时：

1. 从传入的 nonce 起点开始，每次增加 `0x26f5`（9973，素数，利于二进制离散）
2. 调用 `equix.SolveWithNonce(challenge, nonce)`
3. 对每个解计算 `SHA256(challenge || nonce || solution)` 的前 8 字节為 `uint64`
4. 若值 `<= target`，返回该解

验证时：

1. 先计算同一哈希值并检查是否 `<= target`
2. 再调用 `equix.VerifyWithNonce(challenge, nonce, solution)` 确认解结构有效

## 实测数据

来自 `go test ./ratio -run TestSolveCost -v -count=1` 的真实输出：

```text
P=10.0% (expected≈10.000000 tries)
solve  min=56.064083ms avg=189.278854ms max=439.2025ms
verify min=15.542µs avg=24.161µs max=49.709µs
tries  min=2 avg=6.8 max=16
```

这说明在 `P = 0.1` 条件下：

- 求解平均约为 `189ms`
- 验证平均约为 `24µs`
- 实际尝试次数在 `2~16` 之间，平均约 `6.8` 次

> **平台：**
> Mac mini M4 Pro 14 cpu core（10 + 4）

## 说明

`ratio` 包适合需要“按概率调节难度”的场景：例如希望平均每 10 次候选解中约有 1 次命中，就用 `TargetFromProbability(0.1)`。相比固定 bit 目标，它更接近“概率门槛”控制方式，并保留了 Equi-X 官方解校验逻辑。