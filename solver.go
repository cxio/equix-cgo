package equix

import (
	"errors"

	"github.com/cxio/equix/internal/native"
)

type Solver struct {
	ctx *native.Context
}

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

func (s *Solver) SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error) {
	return s.Solve(appendNonce(challenge, nonce))
}

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

func (s *Solver) SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error) {
	return s.SolveWithHashes(appendNonce(challenge, nonce))
}

func (s *Solver) Close() error {
	if s == nil || s.ctx == nil {
		return nil
	}
	s.ctx.Close()
	s.ctx = nil
	return nil
}
