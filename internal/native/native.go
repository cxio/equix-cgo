package native

/*
#include <stdlib.h>
#include <equix.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"
)

var ErrNotSupported = errors.New("equix: requested context type not supported")

type Context struct {
	ptr     *C.equix_ctx
	cleanup runtime.Cleanup
}

func isNotSupp(p *C.equix_ctx) bool {
	return uintptr(unsafe.Pointer(p)) == ^uintptr(0)
}

func alloc(base C.equix_ctx_flags) (*Context, error) {
	v2 := C.equix_ctx_flags(C.EQUIX_V2)
	p := C.equix_alloc(base | v2 | C.EQUIX_CTX_COMPILE)
	if isNotSupp(p) {
		p = C.equix_alloc(base | v2)
	}
	if p == nil {
		return nil, errors.New("equix: out of memory")
	}
	if isNotSupp(p) {
		return nil, ErrNotSupported
	}
	c := &Context{ptr: p}
	c.cleanup = runtime.AddCleanup(c, freeEquixCtx, p)
	return c, nil
}

func NewSolver() (*Context, error) {
	return alloc(C.EQUIX_CTX_SOLVE)
}

func NewVerifier() (*Context, error) {
	return alloc(C.EQUIX_CTX_VERIFY)
}

var freeEquixCount atomic.Uint64

func freeEquixCtx(p *C.equix_ctx) {
	C.equix_free(p)
	freeEquixCount.Add(1)
}

func (c *Context) Close() {
	if c == nil || c.ptr == nil {
		return
	}
	c.cleanup.Stop()
	p := c.ptr
	c.ptr = nil
	freeEquixCtx(p)
	runtime.KeepAlive(c)
}

func (c *Context) dead() error {
	if c == nil || c.ptr == nil {
		return errors.New("equix: nil context")
	}
	return nil
}

func cChallenge(ch []byte) (unsafe.Pointer, C.size_t, func()) {
	if len(ch) == 0 {
		return nil, 0, func() {}
	}
	p := C.CBytes(ch)
	return p, C.size_t(len(ch)), func() { C.free(p) }
}

func (c *Context) Solve(challenge []byte) ([][8]uint16, error) {
	if err := c.dead(); err != nil {
		return nil, err
	}
	defer runtime.KeepAlive(c)
	var out [C.EQUIX_MAX_SOLS]C.equix_solution
	ptr, n, free := cChallenge(challenge)
	defer free()
	got := int(C.equix_solve(c.ptr, ptr, n, (*C.equix_solution)(unsafe.Pointer(&out[0]))))
	if got < 0 || got > int(C.EQUIX_MAX_SOLS) {
		return nil, fmt.Errorf("equix: invalid solution count %d", got)
	}
	sols := make([][8]uint16, got)
	for i := 0; i < got; i++ {
		for j := 0; j < 8; j++ {
			sols[i][j] = uint16(out[i].idx[j])
		}
	}
	return sols, nil
}

func (c *Context) Verify(challenge []byte, idx [8]uint16) (int, error) {
	if err := c.dead(); err != nil {
		return -1, err
	}
	defer runtime.KeepAlive(c)
	var sol C.equix_solution
	for i := 0; i < 8; i++ {
		sol.idx[i] = C.equix_idx(idx[i])
	}
	ptr, n, free := cChallenge(challenge)
	defer free()
	code := int(C.equix_verify(c.ptr, ptr, n, &sol))
	return code, nil
}
