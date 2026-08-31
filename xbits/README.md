# xbits

哈希前缀难度封装。在 Equi-X 解外再套一层 SHA-256 前缀零 bit 门槛，客户端搜 nonce，服务端先滤难度再验 Equi-X。

默认 `TargetBits = 4`，期望约 16 次求解（`2^4`）。Nonce 从 `13` 起，每次加 `0x26f5`（9973，素数，利于二进制离散）。

## 实测

Mac mini M4 Pro 14-core（`TargetBits=4`）：

```text
solve  min=55.293958ms avg=235.674078ms max=627.717458ms
verify min=13.875µs avg=16.973µs max=28.083µs
tries  min=2 avg=8.5 max=23 (TargetBits=4, expected≈16)
```

## 用法

```go
package main

import (
	"fmt"

	"github.com/cxio/equix-cgo/xbits"
)

func main() {
	challenge := []byte("example_challenge_string")

	fmt.Println("Solving puzzle...")
	sol, err := xbits.SolvePuzzle(challenge)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found Solution! Nonce: %d\n", sol.Nonce)

	// 服务端快速验证
	valid := xbits.VerifyPuzzle(challenge, sol)
	fmt.Printf("Verification result: %v\n", valid)
}
```

`PuzzleSolution` 含匹配难度的 `Nonce` 和对应 Equi-X `Solution`。

## 内部计算

**求解**：对每个 nonce 调 `equix.SolveWithNonce`，对每个解算 `SHA-256(challenge || nonce || solution)`，检查前 `TargetBits` 个 bit 是否全 0。

**校验**：先查哈希前缀（快速丢掉无效提交），再 `equix.VerifyWithNonce`。
