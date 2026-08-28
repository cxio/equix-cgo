package equix

const (
	stage1Mask = uint64(1<<15) - 1
	stage2Mask = uint64(1<<30) - 1
	fullMask   = uint64(1<<60) - 1
)

// VerifyHashes 校验 8 个哈希值是否满足 Equi-X 的 Wagner 加法树约束
// （部分和尾零与最终和），纯 Go 实现，不调用 HashWX、不需要 challenge。
//
// 这是合法解的必需但非充分条件：通过只说明这 8 个数满足加法树，
// 不证明它们来自某个 challenge 上的 HashWX。完整 puzzle 校验仍用
// Verify / VerifyWithNonce（只判对错）或 VerifyWithHashes*（对错 + 哈希）。
func VerifyHashes(h Hashes) error {
	pair0 := h[0] + h[1]
	if pair0&stage1Mask != 0 {
		return ErrPartialSum
	}
	pair1 := h[2] + h[3]
	if pair1&stage1Mask != 0 {
		return ErrPartialSum
	}
	pair4 := pair0 + pair1
	if pair4&stage2Mask != 0 {
		return ErrPartialSum
	}
	pair2 := h[4] + h[5]
	if pair2&stage1Mask != 0 {
		return ErrPartialSum
	}
	pair3 := h[6] + h[7]
	if pair3&stage1Mask != 0 {
		return ErrPartialSum
	}
	pair5 := pair2 + pair3
	if pair5&stage2Mask != 0 {
		return ErrPartialSum
	}
	pair6 := pair4 + pair5
	if pair6&fullMask != 0 {
		return ErrFinalSum
	}
	return nil
}
