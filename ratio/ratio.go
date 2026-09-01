package ratio

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/cxio/equix-cgo"
)

// Target 难度门槛阈值
type Target uint64

// Solution 包含成功匹配难度的 Nonce 和对应的 Equi-X 解。
type Solution struct {
	Nonce    uint64
	Solution equix.Solution
}

// TargetFromProbability 根据期望概率 P (0.0 < P <= 1.0) 计算 Target 门槛值
// 例如：
// Equi-X v2 下每个解约需 18ms（Mac mini M4 Pro）。
// - P = 0.1 (1/10 => 10%): 希望共求解 10 个，实测耗时：avg=186ms
// - P = 0.025 (1/40 => 2.5%): 期望共求解 40 个，实测耗时：avg=503ms
// 支持任意微细粒度调整。
func TargetFromProbability(p float64) Target {
	if p <= 0 {
		return 0
	}
	if p >= 1.0 {
		return math.MaxUint64
	}
	return Target(float64(math.MaxUint64) * p)
}

// computeHashValue 计算组合哈希，并将其转为 Target 数值
// 算法处理：Hash(challenge || nonce || solution)[:8] => Target
func computeHashValue(challenge []byte, nonce uint64, sol equix.Solution) (Target, error) {
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

	// 取哈希结果的前 8 字节作为目标数值比较
	return Target(binary.BigEndian.Uint64(hash[:8])), nil
}

// Solve 客户端求解函数 (平均耗时可通过 target 任意线性微调)
// nonce 指定起始 Nonce 值，从该值起每次步进 0x26f5（小素数，利于二进制离散），
// 最终返回的 Nonce 可能不同。多 worker 并行分片时，各起点应避免相差 0x26f5 的整数倍，
// 否则搜索序列完全重叠。
func Solve(challenge []byte, target Target, nonce uint64) (*Solution, error) {
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
				return &Solution{
					Nonce:    nonce,
					Solution: sol,
				}, nil
			}
		}

		nonce += 0x26f5 // 一个素数，二进制离散化
	}
}

// Verify 服务端快速验证函数 (~12us)
func Verify(challenge []byte, target Target, sol *Solution) bool {
	// 1. 优先数值比对 (极速过滤)
	val, err := computeHashValue(challenge, sol.Nonce, sol.Solution)
	if err != nil || val > target {
		return false
	}

	// 2. 验证 Equi-X 解结构 (~12us)
	err = equix.VerifyWithNonce(challenge, sol.Nonce, sol.Solution)
	return err == nil
}
