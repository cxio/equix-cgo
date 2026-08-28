package equix

const (
	stage1Mask = uint64(1<<15) - 1
	stage2Mask = uint64(1<<30) - 1
	fullMask   = uint64(1<<60) - 1
)

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
