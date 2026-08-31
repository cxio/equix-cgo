package ratio

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/cxio/equix-cgo"
)

// PuzzleSolution 包含成功匹配难度的 Nonce 和对应的 Equi-X 解。
// 其中 Nonce 从 13 开始，每次步进值 9973：一个素数，主要考虑二进制离散分布。
type PuzzleSolution struct {
	Nonce    uint64
	Solution equix.Solution
}

// TargetFromProbability 根据期望概率 P (0.0 < P <= 1.0) 计算 Target 门槛值
// 例如：
// Equi-X v2 下每个解约需 18ms（Mac mini M4 Pro 14 core）。
// - P = 0.1 (1/10 => 10%): 希望共求解 10 个，实测耗时：avg=186ms
// - P = 0.025 (1/40 => 2.5%): 期望共求解 40 个，实测耗时：avg=503ms
// 支持任意微细粒度调整。
func TargetFromProbability(p float64) uint64 {
	if p <= 0 {
		return 0
	}
	if p >= 1.0 {
		return math.MaxUint64
	}
	return uint64(float64(math.MaxUint64) * p)
}

// computeHashValue 计算组合哈希，并将其转为 uint64 数值
// 算法处理：Hash(challenge || nonce || solution)[:8] => uint64
func computeHashValue(challenge []byte, nonce uint64, sol equix.Solution) (uint64, error) {
	solBytes, err := sol.MarshalBinary()
	if err != nil {
		return 0, err
	}

	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, nonce)

	h := sha256.New()
	h.Write(challenge)
	h.Write(nonceBytes)
	h.Write(solBytes)
	hash := h.Sum(nil)

	// 取哈希结果的前 8 字节作为 uint64 数值比较
	return binary.BigEndian.Uint64(hash[:8]), nil
}

// SolvePuzzle 客户端求解函数 (平均耗时可通过 target 任意线性微调)
func SolvePuzzle(challenge []byte, target uint64) (*PuzzleSolution, error) {
	var nonce uint64 = 13

	for {
		sols, err := equix.SolveWithNonce(challenge, nonce)
		if err != nil {
			return nil, err
		}

		for _, sol := range sols {
			val, err := computeHashValue(challenge, nonce, sol)
			if err != nil {
				return nil, err
			}

			// 数值比较：只要得到的哈希数值小于 Target 即算通过
			if val <= target {
				return &PuzzleSolution{
					Nonce:    nonce,
					Solution: sol,
				}, nil
			}
		}

		nonce += 0x26f5 // 一个素数，二进制离散化
	}
}

// VerifyPuzzle 服务端快速验证函数 (~12us)
func VerifyPuzzle(challenge []byte, target uint64, puzzleSol *PuzzleSolution) bool {
	// 1. 优先数值比对 (极速过滤)
	val, err := computeHashValue(challenge, puzzleSol.Nonce, puzzleSol.Solution)
	if err != nil || val > target {
		return false
	}

	// 2. 验证 Equi-X 解结构 (~12us)
	err = equix.VerifyWithNonce(challenge, puzzleSol.Nonce, puzzleSol.Solution)
	return err == nil
}
