package equix

import (
	"errors"

	"github.com/cxio/equix/internal/native"
)

type Verifier struct {
	ctx *native.Context
}

func NewVerifier() (*Verifier, error) {
	ctx, err := native.NewVerifier()
	if err != nil {
		if errors.Is(err, native.ErrNotSupported) {
			return nil, ErrNotSupported
		}
		return nil, err
	}
	return &Verifier{ctx: ctx}, nil
}

func (v *Verifier) closed() error {
	if v == nil || v.ctx == nil {
		return ErrClosed
	}
	return nil
}

func (v *Verifier) Verify(challenge []byte, sol Solution) error {
	if err := v.closed(); err != nil {
		return err
	}
	code, err := v.ctx.Verify(challenge, [8]uint16(sol))
	if err != nil {
		return err
	}
	return mapVerify(code)
}

func (v *Verifier) Close() error {
	if v == nil || v.ctx == nil {
		return nil
	}
	v.ctx.Close()
	v.ctx = nil
	return nil
}
