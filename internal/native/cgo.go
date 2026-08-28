package native

/*
#cgo CFLAGS: -std=c11 -O2 -DHASHX_SIZE=8 -DHASHX_STATIC -DHASHWX_STATIC -DEQUIX_STATIC
#cgo CFLAGS: -I${SRCDIR}/../../third_party/equix/include
#cgo CFLAGS: -I${SRCDIR}/../../third_party/equix/src
#cgo CFLAGS: -I${SRCDIR}/../../third_party/hashx/include
#cgo CFLAGS: -I${SRCDIR}/../../third_party/hashx/src
#cgo CFLAGS: -I${SRCDIR}/../../third_party/hashwx/include
#cgo CFLAGS: -I${SRCDIR}/../../third_party/hashwx/src
#cgo windows LDFLAGS: -ladvapi32
#include <stdint.h>
#include <stdlib.h>
#include <equix.h>

void native_make_hashwx(equix_ctx* ctx, const void* challenge, size_t n);
void native_fill_hashes(equix_ctx* ctx, const uint16_t idx[8], uint64_t out[8]);
void native_make_and_fill_hashes(equix_ctx* ctx, const void* challenge, size_t n,
	const uint16_t idx[8], uint64_t out[8]);
*/
import "C"
