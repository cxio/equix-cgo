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

func (s *Solver) Close() error {
	if s == nil || s.ctx == nil {
		return nil
	}
	s.ctx.Close()
	s.ctx = nil
	return nil
}
