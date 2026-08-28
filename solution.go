package equix

import (
	"encoding/binary"
	"fmt"
)

type Solution [8]uint16

type Hashes [8]uint64

type Result struct {
	Solution Solution
	Hashes   Hashes
}

func (s Solution) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint16(b[i*2:], s[i])
	}
	return b, nil
}

func (s *Solution) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("equix: solution must be 16 bytes, got %d", len(data))
	}
	for i := 0; i < 8; i++ {
		s[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	return nil
}

func appendNonce(challenge []byte, nonce uint64) []byte {
	out := make([]byte, len(challenge)+8)
	copy(out, challenge)
	binary.LittleEndian.PutUint64(out[len(challenge):], nonce)
	return out
}
