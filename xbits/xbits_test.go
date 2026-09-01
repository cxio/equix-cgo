package xbits

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

const (
	// 起始 nonce，取小素数。
	nonceStart = 13
	nonceStep  = 0x26f5
)

func TestSolveAndVerify(t *testing.T) {
	challenge := []byte("equix-cgo/xbits cost")

	sol, err := Solve(challenge, nonceStart)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if sol == nil {
		t.Fatal("Solve returned nil solution")
	}
	if !Verify(challenge, sol) {
		t.Fatal("Verify rejected a freshly solved puzzle")
	}

	// 错误 nonce 必须失败。
	bad := *sol
	bad.Nonce++
	if Verify(challenge, &bad) {
		t.Fatal("Verify accepted a mutated nonce")
	}
}

func TestSolveCost(t *testing.T) {
	const samples = 8

	solveDurations := make([]time.Duration, 0, samples)
	verifyDurations := make([]time.Duration, 0, samples)
	nonceTries := make([]uint64, 0, samples)

	for i := range samples {
		challenge := make([]byte, 16)
		binary.LittleEndian.PutUint64(challenge, uint64(i+1))
		binary.LittleEndian.PutUint64(challenge[8:], 0x7821c0de)

		start := time.Now()
		sol, err := Solve(challenge, nonceStart)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("sample %d Solve: %v", i, err)
		}

		vStart := time.Now()
		ok := Verify(challenge, sol)
		vElapsed := time.Since(vStart)
		if !ok {
			t.Fatalf("sample %d Verify failed", i)
		}

		// 尝试次数衡量 Equi-X 求解轮数。
		tries := (sol.Nonce-nonceStart)/nonceStep + 1
		solveDurations = append(solveDurations, elapsed)
		verifyDurations = append(verifyDurations, vElapsed)
		nonceTries = append(nonceTries, tries)

		t.Logf("sample %d: solve=%s verify=%s nonce=%d tries=%d",
			i, elapsed, vElapsed, sol.Nonce, tries)
	}

	t.Logf("solve  min=%s avg=%s max=%s",
		minDuration(solveDurations), avgDuration(solveDurations), maxDuration(solveDurations))
	t.Logf("verify min=%s avg=%s max=%s",
		minDuration(verifyDurations), avgDuration(verifyDurations), maxDuration(verifyDurations))
	t.Logf("tries  min=%d avg=%.1f max=%d (TargetBits=%d, expected≈%d)",
		minUint64(nonceTries), avgUint64(nonceTries), maxUint64(nonceTries),
		TargetBits, 1<<TargetBits)
}

func TestCheckDifficulty(t *testing.T) {
	hash := make([]byte, 32)
	if !checkDifficulty(hash, 4) {
		t.Fatal("all-zero hash should pass 4-bit target")
	}
	hash[0] = 0b00010000
	if checkDifficulty(hash, 4) {
		t.Fatal("bit 4 set should fail 4-bit target")
	}
	hash[0] = 0b00001111
	if !checkDifficulty(hash, 4) {
		t.Fatal("lower 4 bits set should still pass 4-bit target")
	}
}

func minDuration(ds []time.Duration) time.Duration {
	m := ds[0]
	for _, d := range ds[1:] {
		if d < m {
			m = d
		}
	}
	return m
}

func maxDuration(ds []time.Duration) time.Duration {
	m := ds[0]
	for _, d := range ds[1:] {
		if d > m {
			m = d
		}
	}
	return m
}

func avgDuration(ds []time.Duration) time.Duration {
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum / time.Duration(len(ds))
}

func minUint64(vs []uint64) uint64 {
	m := vs[0]
	for _, v := range vs[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxUint64(vs []uint64) uint64 {
	m := vs[0]
	for _, v := range vs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func avgUint64(vs []uint64) float64 {
	var sum uint64
	for _, v := range vs {
		sum += v
	}
	return float64(sum) / float64(len(vs))
}

func BenchmarkSolve(b *testing.B) {
	for i := 0; b.Loop(); i++ {
		challenge := fmt.Appendf(nil, "bench-%d", i)
		if _, err := Solve(challenge, nonceStart); err != nil {
			b.Fatal(err)
		}
	}
}
