package xbits

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

func TestSolveAndVerifyPuzzle(t *testing.T) {
	challenge := []byte("equix-cgo/xbits cost")

	sol, err := SolvePuzzle(challenge)
	if err != nil {
		t.Fatalf("SolvePuzzle: %v", err)
	}
	if sol == nil {
		t.Fatal("SolvePuzzle returned nil solution")
	}
	if !VerifyPuzzle(challenge, sol) {
		t.Fatal("VerifyPuzzle rejected a freshly solved puzzle")
	}

	// 错误 nonce 必须失败。
	bad := *sol
	bad.Nonce++
	if VerifyPuzzle(challenge, &bad) {
		t.Fatal("VerifyPuzzle accepted a mutated nonce")
	}
}

func TestSolvePuzzleCost(t *testing.T) {
	const samples = 8

	solveDurations := make([]time.Duration, 0, samples)
	verifyDurations := make([]time.Duration, 0, samples)
	nonceTries := make([]uint64, 0, samples)

	for i := 0; i < samples; i++ {
		challenge := make([]byte, 16)
		binary.LittleEndian.PutUint64(challenge, uint64(i+1))
		binary.LittleEndian.PutUint64(challenge[8:], 0x7821c0de)

		start := time.Now()
		sol, err := SolvePuzzle(challenge)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("sample %d SolvePuzzle: %v", i, err)
		}

		vStart := time.Now()
		ok := VerifyPuzzle(challenge, sol)
		vElapsed := time.Since(vStart)
		if !ok {
			t.Fatalf("sample %d VerifyPuzzle failed", i)
		}

		// nonce 从 13 起步，步进 0x26f5，用尝试次数衡量 Equi-X 求解轮数。
		tries := (sol.Nonce-13)/0x26f5 + 1
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
	if !checkDifficulty(hash, 5) {
		t.Fatal("all-zero hash should pass 5-bit target")
	}
	hash[0] = 0b00001000
	if checkDifficulty(hash, 5) {
		t.Fatal("bit 5 set should fail 5-bit target")
	}
	hash[0] = 0b00000111
	if !checkDifficulty(hash, 5) {
		t.Fatal("only lower 3 bits set should still pass 5-bit target")
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

func BenchmarkSolvePuzzle(b *testing.B) {
	for i := 0; b.Loop(); i++ {
		challenge := fmt.Appendf(nil, "bench-%d", i)
		if _, err := SolvePuzzle(challenge); err != nil {
			b.Fatal(err)
		}
	}
}
