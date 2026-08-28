package equix

import (
	"encoding/binary"
	"fmt"
)

// Solution 表示一个 Equi-X 解，即 8 个 16 位索引。
// 对应 C 的 equix_solution.idx[8]；一次求解最多返回 8 个解。
type Solution [8]uint16

// Hashes 是一个解对应的 8 个 HashWX 输出（完整 64 位，不截断到 60 位）。
// 与 Solution 的下标一一对应：Hashes[i] = HashWX(seed, Solution[i])。
type Hashes [8]uint64

// Result 将一个解与其对应的 8 个哈希值捆绑在一起。
type Result struct {
	// Solution 是解本身。
	Solution Solution
	// Hashes 是 Solution 各索引对应的 HashWX 输出。
	Hashes Hashes
}

// MarshalBinary 将解编码为 16 字节 little-endian（idx[0]…idx[7]）。
func (s Solution) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)
	for i := 0; i < 8; i++ {
		binary.LittleEndian.PutUint16(b[i*2:], s[i])
	}
	return b, nil
}

// UnmarshalBinary 从 16 字节 little-endian 数据解码解。
// 长度不为 16 时返回错误。
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
