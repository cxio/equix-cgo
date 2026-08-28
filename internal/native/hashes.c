#include <stdint.h>
#include <string.h>
#include <equix.h>
#include <hashwx.h>
#include "context.h"
#include "blake2.h"

extern const blake2b_param equix_v2_blake2_params;

void native_make_hashwx(equix_ctx* ctx, const void* challenge, size_t n) {
	uint8_t seed[HASHWX_SEED_SIZE];
	blake2b_state st;
	hashx_blake2b_init_param(&st, &equix_v2_blake2_params);
	hashx_blake2b_update(&st, challenge, n);
	hashx_blake2b_final(&st, seed, HASHWX_SEED_SIZE);
	hashwx_make(ctx->hash_v2, seed);
}

void native_fill_hashes(equix_ctx* ctx, const uint16_t idx[8], uint64_t out[8]) {
	int i;
	for (i = 0; i < 8; i++) {
		out[i] = hashwx_exec(ctx->hash_v2, idx[i]);
	}
}

void native_make_and_fill_hashes(equix_ctx* ctx, const void* challenge, size_t n,
	const uint16_t idx[8], uint64_t out[8]) {
	native_make_hashwx(ctx, challenge, n);
	native_fill_hashes(ctx, idx, out);
}
