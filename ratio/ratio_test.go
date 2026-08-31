package ratio

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
)

const (
	// 期望通过概率 10%，约需 10 个 Equi-X 解。
	testProbability = 0.1
	costSamples     = 8
	nonceStart      = 13
	nonceStep       = 0x26f5
)

func TestTargetFromProbability(t *testing.T) {
	if got := TargetFromProbability(0); got != 0 {
		t.Fatalf("P<=0: got %d, want 0", got)
	}
	if got := TargetFromProbability(1); got != math.MaxUint64 {
		t.Fatalf("P>=1: got %d, want MaxUint64", got)
	}

	got := TargetFromProbability(testProbability)
	want := uint64(float64(math.MaxUint64) * testProbability)
	if got != want {
		t.Fatalf("P=%.3f: got %d, want %d", testProbability, got, want)
	}
}

func TestSolveAndVerifyPuzzle(t *testing.T) {
	challenge := []byte("equix-cgo/ratiox cost")
	target := TargetFromProbability(testProbability)

	sol, err := SolvePuzzle(challenge, target)
	if err != nil {
		t.Fatalf("SolvePuzzle: %v", err)
	}
	if sol == nil {
		t.Fatal("SolvePuzzle returned nil solution")
	}
	if !VerifyPuzzle(challenge, target, sol) {
		t.Fatal("VerifyPuzzle rejected a freshly solved puzzle")
	}

	// 错误 nonce 必须失败。
	bad := *sol
	bad.Nonce++
	if VerifyPuzzle(challenge, target, &bad) {
		t.Fatal("VerifyPuzzle accepted a mutated nonce")
	}
}

func TestSolvePuzzleCost(t *testing.T) {
	target := TargetFromProbability(testProbability)

	solveDurations := make([]time.Duration, 0, costSamples)
	verifyDurations := make([]time.Duration, 0, costSamples)
	nonceTries := make([]uint64, 0, costSamples)

	for i := 0; i < costSamples; i++ {
		challenge := make([]byte, 16)
		binary.LittleEndian.PutUint64(challenge, uint64(i+1))
		binary.LittleEndian.PutUint64(challenge[8:], 0x7821c0de)

		start := time.Now()
		sol, err := SolvePuzzle(challenge, target)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("sample %d SolvePuzzle: %v", i, err)
		}

		vStart := time.Now()
		ok := VerifyPuzzle(challenge, target, sol)
		vElapsed := time.Since(vStart)
		if !ok {
			t.Fatalf("sample %d VerifyPuzzle failed", i)
		}

		// nonce 从 13 起步，步进 0x26f5，用尝试次数衡量 Equi-X 求解轮数。
		tries := (sol.Nonce-nonceStart)/nonceStep + 1
		solveDurations = append(solveDurations, elapsed)
		verifyDurations = append(verifyDurations, vElapsed)
		nonceTries = append(nonceTries, tries)

		t.Logf("sample %d: solve=%s verify=%s nonce=%d tries=%d",
			i, elapsed, vElapsed, sol.Nonce, tries)
	}

	t.Logf("P=%.1f%% (expected≈%f tries)", testProbability*100, math.Floor(1/testProbability))
	t.Logf("solve  min=%s avg=%s max=%s",
		minDuration(solveDurations), avgDuration(solveDurations), maxDuration(solveDurations))
	t.Logf("verify min=%s avg=%s max=%s",
		minDuration(verifyDurations), avgDuration(verifyDurations), maxDuration(verifyDurations))
	t.Logf("tries  min=%d avg=%.1f max=%d",
		minUint64(nonceTries), avgUint64(nonceTries), maxUint64(nonceTries))
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
