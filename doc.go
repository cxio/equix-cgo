// Package equix 封装了 tevador/equix 的 Equi-X v2 算法（HashWX），
// 提供求解与校验能力。
//
// 使用方式有两种：
//
//   - 包级函数 Solve / Verify 及其带 nonce、带哈希的变体，内部通过
//     sync.Pool 复用上下文，可从多个 goroutine 并发调用；
//   - 可复用的 Solver / Verifier 实例，适合反复求解或校验，但同一
//     实例不是线程安全的，用完必须调用 Close。
//
// 求解输入为 challenge（可为 nil 或空切片）；带 nonce 的变体在
// challenge 后追加 8 字节 little-endian uint64 nonce，本包不搜索或
// 递增 nonce。一次求解最多返回 8 个解。
//
// 使用本包需要 CGO_ENABLED=1 和 C 编译器（Windows 用 MinGW）。
// 静态链入的官方 C 代码为 LGPL-3.0，分发二进制时须遵循其许可要求。
package equix
