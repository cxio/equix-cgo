package equix

import "errors"

// 本包可通过 errors.Is 判断的哨兵错误。
var (
	// ErrNotSupported 表示编译型（JIT）与解释型 context 均无法分配。
	// 参见 NewSolver / NewVerifier。
	ErrNotSupported = errors.New("equix: requested context type not supported")
	// ErrChallenge 表示 challenge 无效（上游 C 的 EQUIX_CHALLENGE）。
	// 在 Equi-X v2 路径上几乎不会出现。
	ErrChallenge = errors.New("equix: invalid challenge")
	// ErrOrder 表示解的索引顺序不符合要求。
	ErrOrder = errors.New("equix: indices not in required order")
	// ErrPartialSum 表示部分和缺少规定数量的尾零。
	ErrPartialSum = errors.New("equix: partial sum missing trailing zeros")
	// ErrFinalSum 表示八个哈希之和的低 60 位不为 0。
	ErrFinalSum = errors.New("equix: hashes do not sum to zero")
	// ErrClosed 表示使用了已 Close 的 Solver 或 Verifier。
	ErrClosed = errors.New("equix: use of closed solver or verifier")
)
