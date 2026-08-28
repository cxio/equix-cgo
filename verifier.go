package equix

import (
	"errors"

	"github.com/cxio/equix/internal/native"
)

// Verifier 表示一个可复用的 Equi-X 校验上下文。
// 同一实例不是线程安全的，用完必须调用 Close。
type Verifier struct {
	ctx *native.Context
}

// NewVerifier 分配一个校验上下文，优先编译型 HashWX（JIT），
// 失败则回退到解释型。
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

// Verify 校验 challenge 下的解，合法返回 nil。
// 失败时可用 errors.Is 区分 ErrOrder / ErrPartialSum / ErrFinalSum。
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

// VerifyWithNonce 等价于 Verify(append(challenge, little-endian(nonce)...), sol)。
func (v *Verifier) VerifyWithNonce(challenge []byte, nonce uint64, sol Solution) error {
	return v.Verify(appendNonce(challenge, nonce), sol)
}

// VerifyWithHashes 校验解并同时返回其 8 个 HashWX 哈希值。
// ErrOrder / ErrPartialSum / ErrFinalSum 时仍返回已算出的哈希；
// 只有 ErrClosed 或分配失败才返回零值 Hashes。
func (v *Verifier) VerifyWithHashes(challenge []byte, sol Solution) (Hashes, error) {
	var zero Hashes
	if err := v.closed(); err != nil {
		return zero, err
	}
	h, code, err := v.ctx.VerifyWithHashes(challenge, [8]uint16(sol))
	if err != nil {
		return zero, err
	}
	return Hashes(h), mapVerify(code)
}

// VerifyWithHashesAndNonce 是 VerifyWithHashes 的带 nonce 变体。
func (v *Verifier) VerifyWithHashesAndNonce(challenge []byte, nonce uint64, sol Solution) (Hashes, error) {
	return v.VerifyWithHashes(appendNonce(challenge, nonce), sol)
}

// Close 释放底层 C 上下文。幂等，对 nil 接收者安全；
// Close 之后再调用任何方法会返回 ErrClosed。
func (v *Verifier) Close() error {
	if v == nil || v.ctx == nil {
		return nil
	}
	v.ctx.Close()
	v.ctx = nil
	return nil
}
