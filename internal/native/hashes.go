package native

/*
#include <stdint.h>
#include <equix.h>

void native_make_hashwx(equix_ctx* ctx, const void* challenge, size_t n);
void native_fill_hashes(equix_ctx* ctx, const uint16_t idx[8], uint64_t out[8]);
void native_make_and_fill_hashes(equix_ctx* ctx, const void* challenge, size_t n,
	const uint16_t idx[8], uint64_t out[8]);
*/
import "C"

func (c *Context) FillHashes(idx [8]uint16) ([8]uint64, error) {
	var zero [8]uint64
	if err := c.dead(); err != nil {
		return zero, err
	}
	var cidx [8]C.uint16_t
	var out [8]C.uint64_t
	for i := 0; i < 8; i++ {
		cidx[i] = C.uint16_t(idx[i])
	}
	C.native_fill_hashes(c.ptr, &cidx[0], &out[0])
	var h [8]uint64
	for i := 0; i < 8; i++ {
		h[i] = uint64(out[i])
	}
	return h, nil
}

func (c *Context) MakeAndFillHashes(challenge []byte, idx [8]uint16) ([8]uint64, error) {
	var zero [8]uint64
	if err := c.dead(); err != nil {
		return zero, err
	}
	var cidx [8]C.uint16_t
	var out [8]C.uint64_t
	for i := 0; i < 8; i++ {
		cidx[i] = C.uint16_t(idx[i])
	}
	ptr, n, free := cChallenge(challenge)
	defer free()
	C.native_make_and_fill_hashes(c.ptr, ptr, n, &cidx[0], &out[0])
	var h [8]uint64
	for i := 0; i < 8; i++ {
		h[i] = uint64(out[i])
	}
	return h, nil
}

func (c *Context) SolveWithHashes(challenge []byte) ([][8]uint16, [][8]uint64, error) {
	sols, err := c.Solve(challenge)
	if err != nil {
		return nil, nil, err
	}
	hs := make([][8]uint64, len(sols))
	for i := range sols {
		hs[i], err = c.FillHashes(sols[i])
		if err != nil {
			return nil, nil, err
		}
	}
	return sols, hs, nil
}
