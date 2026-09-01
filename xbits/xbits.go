package xbits

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/cxio/equix-cgo"
)

// TargetBits 设定为 4 bits，
// 期望尝试求解约 16 个，耗时约 235ms（Mac mini M4 Pro）。
const TargetBits = 4

// Solution 包含成功匹配难度的 Nonce 和对应的 Equi-X 解。
type Solution struct {
	Nonce    uint64
	Solution equix.Solution
}

// checkDifficulty 检查 SHA-256 哈希值的前 N 个 bit 是否全为 0
func checkDifficulty(hash []byte, bits int) bool {
	fullBytes := bits / 8
	for i := range fullBytes {
		if hash[i] != 0 {
			return false
		}
	}
	remainingBits := bits % 8
	if remainingBits > 0 {
		mask := byte(0xFF << (8 - remainingBits))
		if (hash[fullBytes] & mask) != 0 {
			return false
		}
	}
	return true
}

// computeCombinedHash 计算 (challenge || nonce || solution) 的 SHA-256 哈希
func computeCombinedHash(challenge []byte, nonce uint64, sol equix.Solution) ([]byte, error) {
	solBytes, err := sol.MarshalBinary()
	if err != nil {
		return nil, err
	}

	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)

	h := sha256.New()
	h.Write(challenge)
	h.Write(nonceBytes)
	h.Write(solBytes)

	return h.Sum(nil), nil
}

// Solve 客户端求解函数 (平均耗时 ~240ms)
// nonce 指定起始 Nonce 值，从该值起每次步进 0x26f5（小素数，利于二进制离散），
// 最终返回的 Nonce 可能不同。多 worker 并行分片时，各起点应避免相差 0x26f5 的整数倍，
// 否则搜索序列完全重叠。
func Solve(challenge []byte, nonce uint64) (*Solution, error) {
	for {
		// 1. 调用 Go 封装接口寻找当前 Nonce 下的 Equi-X 解
		sols, err := equix.SolveWithNonce(challenge, nonce)
		if err != nil {
			return nil, err
		}

		// 2. 遍历当前 Nonce 产出的解，校验外层 Difficulty Target
		for _, sol := range sols {
			combinedHash, err := computeCombinedHash(challenge, nonce, sol)
			if err != nil {
				return nil, err
			}

			if checkDifficulty(combinedHash, TargetBits) {
				return &Solution{
					Nonce:    nonce,
					Solution: sol,
				}, nil
			}
		}

		nonce += 0x26f5 // 一个素数，二进制离散化
	}
}

// Verify 服务端验证函数 (验证仅需 ~14us + SHA256 开销)
func Verify(challenge []byte, sol *Solution) bool {
	// 1. 优先校验外层难度前缀 (快速过滤无效提交)
	combinedHash, err := computeCombinedHash(challenge, sol.Nonce, sol.Solution)
	if err != nil || !checkDifficulty(combinedHash, TargetBits) {
		return false
	}

	// 2. 校验核心 Equi-X 解结构
	err = equix.VerifyWithNonce(challenge, sol.Nonce, sol.Solution)
	return err == nil
}
