// Package puzz 在 Equi-X 求解之上叠加一层可调难度的 SHA-256 门槛，
// 提供“客户端搜索、服务端验证”的 PoW 谜题封装。
//
// 难度由 [Threshold] 表达：对每个候选解计算
//
//	SHA256(challenge || nonce || solution) 的前 8 字节（大端 uint64）
//
// 该值小于等于 Threshold 即命中。两种构造方式对应同一机制：
//
//   - [FromProbability]：按期望命中概率构造（如 0.1 表示约 10% 的候选解命中）；
//   - [FromBits]：按组合哈希前缀零位数构造（如 4 位等价于 6.25% 命中概率）。
//
// [Solve] 从指定 nonce 起步、以素数 0x26f5 步进反复求解，直到命中难度。
// 搜索没有天然终点，需要限时或取消时使用 [SolveContext]。
//
// [Verify] 先做廉价的哈希阈值比对（约百纳秒），通过后再做完整的 Equi-X
// 结构校验（约 10µs 量级），适合服务端对不可信提交的快速过滤。
//
// 需要同时得到解对应的 8 个 HashWX 哈希时，使用 [SolveWithHashes]、
// [SolveContextWithHashes] 与 [VerifyWithHashes]。哈希类型为
// [github.com/cxio/equix-cgo.Hashes]，作为多返回值，不进入 Solution 编码。
//
// 命中期望需要约 1/p（或 2^bits）个 Equi-X 候选解；Equi-X 每个 nonce 平均
// 产出约 1.7 个候选解，因此折合约 0.6/p（或 2^bits/1.7）轮 nonce 尝试。
//
// 所有函数内部经主包的池化入口工作，可从多个 goroutine 并发调用。
// 使用本包需要 CGO_ENABLED=1 和 C 编译器（与主包一致）。
package puzz
