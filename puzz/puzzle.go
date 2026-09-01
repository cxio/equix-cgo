package puzz

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cxio/equix-cgo"
)

// DefaultBits 是文档示例采用的默认前缀零位数，命中概率约 2^-4 = 6.25%。
const DefaultBits = 4

// nonceStep 是 Solve 的 nonce 步进量：9973，一个与 2^64 互素的奇素数。
// 选用奇素数可避免多 worker 的搜索序列与 2 的幂步进对齐而互相重叠；
// 但各 worker 的起点仍应避免相差 0x26f5 的整数倍，否则序列完全重合。
const nonceStep uint64 = 0x26f5

// Threshold 是难度阈值：组合哈希 SHA256(challenge || nonce || solution)
// 的前 8 字节按大端解读后，数值小于等于 Threshold 即命中。
type Threshold uint64

// FromProbability 返回使每个候选解命中概率约为 p 的阈值。
// 命中概率的精确值为 (th+1)/2^64，与 p 的偏差不超过 2^-53（IEEE double 舍入）。
// p 必须落在 (0, 1] 区间且不小于 2^-53（低于该值无法用 64 位阈值表达，
// 会退化为实际不可达的 0），NaN 或越界返回错误。
func FromProbability(p float64) (Threshold, error) {
	if math.IsNaN(p) || p <= 0 || p > 1 {
		return 0, fmt.Errorf("puzzle: probability must be in (0, 1], got %g", p)
	}
	if p == 1 {
		return math.MaxUint64, nil
	}
	th := Threshold(float64(math.MaxUint64) * p)
	if th == 0 {
		return 0, fmt.Errorf("puzzle: probability %g is below the 2^-64 representable minimum", p)
	}
	return th, nil
}

// FromBits 返回“组合哈希前 bits 位全零”对应的阈值 MaxUint64 >> bits，
// 命中概率精确为 2^-bits。bits 必须在 [0, 63]；bits=64 时阈值为 0，
// 实际不可达，因此越界或为负一律返回错误。
func FromBits(bits int) (Threshold, error) {
	if bits < 0 || bits > 63 {
		return 0, fmt.Errorf("puzzle: bits must be in [0, 63], got %d", bits)
	}
	return Threshold(uint64(math.MaxUint64) >> bits), nil
}

// hit 判断组合哈希的大端前 8 字节是否不超过阈值。
func (th Threshold) hit(hash *[32]byte) bool {
	return binary.BigEndian.Uint64(hash[:8]) <= uint64(th)
}

// combinedHash 计算 SHA256(challenge || nonce || solution) 并写入 dst。
// 拼接的后缀（8 字节 nonce + 16 字节解）为定长，因此不同的
// (challenge, nonce, solution) 三元组不会产生相同的输入流，无编码歧义。
// nonce 采用 little-endian 编码，与主包 Equi-X 的输入约定一致；
// 阈值比较按大端解读哈希字节，仅是“把前 8 字节当作大数”的约定。
func combinedHash(dst *[32]byte, challenge []byte, nonce uint64, sol equix.Solution) {
	var tail [24]byte
	binary.LittleEndian.PutUint64(tail[:8], nonce)
	for i, v := range sol {
		binary.LittleEndian.PutUint16(tail[8+2*i:], v)
	}
	h := sha256.New()
	h.Write(challenge)
	h.Write(tail[:])
	h.Sum(dst[:0])
}

// Solution 是命中难度的完整答案：搜索时使用的 Nonce 与该 nonce 下的 Equi-X 解。
type Solution struct {
	Nonce    uint64
	Solution equix.Solution
}

// MarshalBinary 编码为 24 字节：8 字节 little-endian Nonce，
// 后接 16 字节解：idx[0]…idx[7]，各 2 字节 little-endian。
func (s Solution) MarshalBinary() ([]byte, error) {
	solBytes, err := s.Solution.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 24)
	binary.LittleEndian.PutUint64(out, s.Nonce)
	copy(out[8:], solBytes)
	return out, nil
}

// UnmarshalBinary 解码 MarshalBinary 的输出，要求恰好 24 字节。
func (s *Solution) UnmarshalBinary(data []byte) error {
	if len(data) != 24 {
		return fmt.Errorf("puzzle: solution must be 24 bytes, got %d", len(data))
	}
	if err := s.Solution.UnmarshalBinary(data[8:]); err != nil {
		return err
	}
	s.Nonce = binary.LittleEndian.Uint64(data[:8])
	return nil
}

// Solve 从 nonce 起步搜索命中难度的解：每个 nonce 调用一次 Equi-X 求解，
// 对每个候选解做阈值判定，未命中则 nonce 前进 nonceStep 后重试。
// 搜索没有天然终点（阈值越低期望耗时越长），需要限时或取消机制时可用 SolveContext。
func Solve(challenge []byte, th Threshold, nonce uint64) (*Solution, error) {
	return SolveContext(context.Background(), challenge, th, nonce)
}

// SolveContext 是带取消的 Solve：进入时及每轮求解（约数十毫秒）之间检查
// ctx，取消时返回 ctx.Err()（context.Canceled 或 context.DeadlineExceeded）。
// 取消最多延迟一轮生效；若当前轮已产出命中解，仍优先返回该有效结果。
func SolveContext(ctx context.Context, challenge []byte, th Threshold, nonce uint64) (*Solution, error) {
	var hash [32]byte
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sols, err := equix.SolveWithNonce(challenge, nonce)
		if err != nil {
			return nil, err
		}
		for _, sol := range sols {
			combinedHash(&hash, challenge, nonce, sol)
			if th.hit(&hash) {
				return &Solution{Nonce: nonce, Solution: sol}, nil
			}
		}
		nonce += nonceStep
	}
}

// Verify 校验提交的解：先做廉价的阈值比对过滤（一次 SHA-256，约百纳秒），
// 通过后再做完整的 Equi-X 结构校验（约 10µs 量级），无效提交的平均验证
// 成本因此接近一次哈希。sol 为 nil 或任一检查不通过时返回 false。
func Verify(challenge []byte, th Threshold, sol *Solution) bool {
	if sol == nil {
		return false
	}
	var hash [32]byte
	combinedHash(&hash, challenge, sol.Nonce, sol.Solution)
	if !th.hit(&hash) {
		return false
	}
	return equix.VerifyWithNonce(challenge, sol.Nonce, sol.Solution) == nil
}
