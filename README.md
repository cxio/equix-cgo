# equix

## 概要

Go cgo 封装，绑定 [tevador/equix](https://github.com/tevador/equix) 的 **equix_v2** 分支（HashWX）。

    import "github.com/cxio/equix"

    sols, err := equix.SolveWithNonce(challenge, nonce)
    if err != nil {
        return err
    }
    for _, sol := range sols {
        h, err := equix.VerifyWithHashesAndNonce(challenge, nonce, sol)
        if err != nil {
            return err
        }
        _ = h
    }

需要 `CGO_ENABLED=1` 和 C 编译器（gcc/clang；Windows 用 MinGW）。不支持 `CGO_ENABLED=0`。

钉死的上游 commit：

- equix `350a85dedda1344637dac09a1de786ee63a5fb01`（`equix_v2`）
- hashx `08babdf4f41b0b8991d1fa94914c7c6902de0cb6`
- hashwx `d771cbf6cdc070755f7d137cdcf9d781af14da3f`


## 用例

包级函数（`Solve` / `Verify` 及带 nonce、带哈希的变体）内部用 `sync.Pool` 复用 context，**可以**从多个 goroutine 同时调用。需要反复求解或校验时，也可以自己持有 `Solver` / `Verifier`，避免每次从池里取；同一个实例**不是**线程安全的，用完必须 `Close`（solver 约 1.8 MiB C 堆）。

一次 `Solve*` 最多返回 8 个解。部分 challenge 本来无解：成功时仍返回空切片和 `nil` error，不会返回 `nil` 切片。

### 包级：challenge + nonce

Equi-X 输入是 `challenge` 后接 8 字节 little-endian `uint64` nonce。本包**不**搜索或递增 nonce；由调用方改 `challenge` 或 `nonce`。`nonce == 0` 合法。

```go
package main

import (
    "fmt"

    "github.com/cxio/equix"
)

func main() {
    challenge := []byte("cxio")
    var nonce uint64
    var sols []equix.Solution
    var err error
    for n := uint64(0); n < 32 && len(sols) == 0; n++ {
        nonce = n
        sols, err = equix.SolveWithNonce(challenge, nonce)
        if err != nil {
            panic(err)
        }
    }
    if len(sols) == 0 {
        fmt.Println("no solution in this nonce range")
        return
    }
    for _, sol := range sols {
        if err := equix.VerifyWithNonce(challenge, nonce, sol); err != nil {
            panic(err)
        }
    }
    fmt.Printf("nonce=%d solutions=%d first=%v\n", nonce, len(sols), sols[0])
}
```

不附带那 8 字节时用 `Solve` / `Verify`：输入就是 `challenge` 本身（可为 `nil` 或空切片）。

`SolveWithNonce(ch, n)` 与 `Solve(append(ch, little-endian uint64(n)...))` 的解集相同。

### 复用 Solver / Verifier

同一实例上串行多次调用；并行时各 goroutine 各建一个。

```go
s, err := equix.NewSolver()
if err != nil {
    return err
}
defer s.Close()

v, err := equix.NewVerifier()
if err != nil {
    return err
}
defer v.Close()

for _, ch := range challenges {
    sols, err := s.Solve(ch)
    if err != nil {
        return err
    }
    for _, sol := range sols {
        if err := v.Verify(ch, sol); err != nil {
            return err
        }
    }
}
```

`Close` 对 `nil` 接收者和重复调用都安全。Close 后再 `Solve` / `Verify` 会得到 `ErrClosed`。

### 带 HashWX 的求解与校验

每个解对应 8 个 `uint64`：`Hashes[i] = HashWX(seed, Solution[i])`（完整 64 位，不是截到 60 位）。合法解满足 `Hashes[0]+…+Hashes[7] ≡ 0 (mod 2^60)`。

```go
results, err := equix.SolveWithHashesAndNonce(challenge, nonce)
if err != nil {
    return err
}
for _, r := range results {
    if err := equix.VerifyHashes(r.Hashes); err != nil {
        return err // 代数树不成立
    }
    h, err := equix.VerifyWithHashesAndNonce(challenge, nonce, r.Solution)
    if err != nil {
        return err // 顺序 / 部分和 / 最终和未通过官方校验
    }
    _ = h // 与 r.Hashes 相同
}
```

`VerifyWithHashes*` 会先生成 HashWX 再做与 `Verify` 相同的检查。因此 **`ErrOrder` / `ErrPartialSum` / `ErrFinalSum` 时仍返回已算出的哈希**；只有 `ErrClosed` 或分配失败才是零值 `Hashes`。

`VerifyHashes` 是纯 Go，不调用 HashWX、不需要 challenge。它是合法解的必要但非充分条件：通过只说明这 8 个数满足加法树，不证明它们来自某个 challenge。完整 puzzle 仍用 `Verify*` 或 `VerifyWithHashes*`。

无 nonce 的对应 API：`SolveWithHashes`、`VerifyWithHashes`。

### 解的 16 字节编码

`Solution` 是 `[8]uint16`，对应 C 的 `equix_solution.idx[8]`。`MarshalBinary` 写出 16 字节 little-endian（`idx[0]`…`idx[7]`）；`UnmarshalBinary` 要求恰好 16 字节。

```go
b, err := sol.MarshalBinary()
if err != nil {
    return err
}
var sol2 equix.Solution
if err := sol2.UnmarshalBinary(b); err != nil {
    return err
}
```

`Hashes` 没有二进制编解码。

### 错误

可用 `errors.Is` 判断：

| 值 | 何时 |
|----|------|
| `ErrNotSupported` | JIT 与解释器都分配不了 context |
| `ErrChallenge` | C `EQUIX_CHALLENGE`（v2 上几乎不会出现） |
| `ErrOrder` | 索引顺序不对 |
| `ErrPartialSum` | 部分和缺少规定的尾零 |
| `ErrFinalSum` | 八哈希之和低 60 位不为 0 |
| `ErrClosed` | 使用已 Close 的 `Solver` / `Verifier` |

`UnmarshalBinary` 长度错误、内存分配失败等是普通 `error`，不是上表哨兵。

## 许可

本仓库 Go 代码为 MIT。`third_party/equix`、`third_party/hashx`、`third_party/hashwx` 为 LGPL-3.0。静态链入官方 C 构成 LGPL 组合作品；分发二进制时须保留 LGPL 声明，并提供这些 C 源码（本仓库已包含）。
