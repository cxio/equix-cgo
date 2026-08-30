package xbits

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/cxio/equix-cgo"
)

// TargetBits 设定为 5 bits，
// 期望尝试约 32 次求解，耗时约 80ms~100ms
// 注：暂不支持定制位数/难度。
const TargetBits = 5

// PuzzleSolution 包含成功匹配难度的 Nonce 和对应的 Equi-X 解。
// 其中 Nonce 从 13 开始，每次步进值 9973：一个素数，主要考虑二进制离散分布。
type PuzzleSolution struct {
	Nonce    uint64
	Solution equix.Solution
}

// checkDifficulty 检查 SHA-256 哈希值的前 N 个 bit 是否全为 0
func checkDifficulty(hash []byte, bits int) bool {
	fullBytes := bits / 8
	for i := 0; i < fullBytes; i++ {
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

// SolvePuzzle 客户端求解函数 (平均耗时 ~80ms)
func SolvePuzzle(challenge []byte) (*PuzzleSolution, error) {
	var nonce uint64 = 13

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
				return &PuzzleSolution{
					Nonce:    nonce,
					Solution: sol,
				}, nil
			}
		}

		nonce += 0x26f5 // 一个素数，二进制离散化
	}
}

// VerifyPuzzle 服务端验证函数 (验证仅需 ~12us + SHA256 开销)
func VerifyPuzzle(challenge []byte, puzzleSol *PuzzleSolution) bool {
	// 1. 优先校验外层难度前缀 (快速过滤无效提交)
	combinedHash, err := computeCombinedHash(challenge, puzzleSol.Nonce, puzzleSol.Solution)
	if err != nil || !checkDifficulty(combinedHash, TargetBits) {
		return false
	}

	// 2. 校验核心 Equi-X 解结构
	err = equix.VerifyWithNonce(challenge, puzzleSol.Nonce, puzzleSol.Solution)
	return err == nil
}
