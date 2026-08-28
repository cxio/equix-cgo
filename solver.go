package equix

import (
	"errors"

	"github.com/cxio/equix/internal/native"
)

// Solver 表示一个可复用的 Equi-X 求解上下文。
// 同一实例不是线程安全的，用完必须调用 Close。
type Solver struct {
	ctx *native.Context
}

// NewSolver 分配一个求解上下文，优先编译型 HashWX（JIT），
// 失败则回退到解释型。
func NewSolver() (*Solver, error) {
	ctx, err := native.NewSolver()
	if err != nil {
		if errors.Is(err, native.ErrNotSupported) {
			return nil, ErrNotSupported
		}
		return nil, err
	}
	return &Solver{ctx: ctx}, nil
}

func (s *Solver) closed() error {
	if s == nil || s.ctx == nil {
		return ErrClosed
	}
	return nil
}

// Solve 求解 challenge（可为 nil 或空切片），最多返回 8 个解。
// 无解时返回空切片（非 nil）和 nil error。
func (s *Solver) Solve(challenge []byte) ([]Solution, error) {
	if err := s.closed(); err != nil {
		return nil, err
	}
	raw, err := s.ctx.Solve(challenge)
	if err != nil {
		return nil, err
	}
	return toSolutions(raw), nil
}

// SolveWithNonce 等价于 Solve(append(challenge, little-endian(nonce)...))。
// 本方法不搜索或递增 nonce，nonce 由调用方确定。
func (s *Solver) SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error) {
	return s.Solve(appendNonce(challenge, nonce))
}

// SolveWithHashes 在 Solve 的基础上，额外返回每个解对应的 8 个 HashWX 哈希值。
// 其 .Solution 序列与 Solve 相同。
func (s *Solver) SolveWithHashes(challenge []byte) ([]Result, error) {
	if err := s.closed(); err != nil {
		return nil, err
	}
	idx, hs, err := s.ctx.SolveWithHashes(challenge)
	if err != nil {
		return nil, err
	}
	out := make([]Result, len(idx))
	for i := range idx {
		out[i] = Result{Solution: Solution(idx[i]), Hashes: Hashes(hs[i])}
	}
	return out, nil
}

// SolveWithHashesAndNonce 是 SolveWithHashes 的带 nonce 变体。
func (s *Solver) SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error) {
	return s.SolveWithHashes(appendNonce(challenge, nonce))
}

// Close 释放底层 C 上下文。幂等，对 nil 接收者安全；
// Close 之后再调用任何方法会返回 ErrClosed。
func (s *Solver) Close() error {
	if s == nil || s.ctx == nil {
		return nil
	}
	s.ctx.Close()
	s.ctx = nil
	return nil
}
