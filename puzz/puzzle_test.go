package puzz

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/cxio/equix-cgo"
)

// 测试用起始 nonce，取小素数；步进量直接复用包级 nonceStep。
const nonceStart = 13

func TestFromProbability(t *testing.T) {
	// 非法输入必须报错，不得静默产出不可达阈值。
	for _, p := range []float64{0, -1, math.NaN(), 1.5} {
		if _, err := FromProbability(p); err == nil {
			t.Errorf("FromProbability(%v) should fail", p)
		}
	}

	// p = 1 恒命中。
	if th, err := FromProbability(1); err != nil || th != math.MaxUint64 {
		t.Errorf("FromProbability(1) = %d, %v; want MaxUint64, nil", th, err)
	}

	// 2 的幂概率可独立推导精确值：命中概率约 p，对应阈值约 2^64·p。
	for _, tc := range []struct {
		p    float64
		want uint64
	}{
		{0.5, 1 << 63},
		{0.25, 1 << 62},
	} {
		th, err := FromProbability(tc.p)
		if err != nil || uint64(th) != tc.want {
			t.Errorf("FromProbability(%v) = %d, %v; want %d, nil", tc.p, th, err, tc.want)
		}
	}

	// 一般值只断言语义（命中概率 ≈ p），不锁实现公式。
	// 实际偏差来自浮点舍入（约 2^-64 相对量），远小于 1e-12。
	const two64 = float64(1<<63) * 2
	for _, p := range []float64{0.1, 0.025, 1e-6, 0.999} {
		th, err := FromProbability(p)
		if err != nil {
			t.Fatalf("FromProbability(%v): %v", p, err)
		}
		got := (float64(uint64(th)) + 1) / two64
		if math.Abs(got-p) > 1e-12 {
			t.Errorf("FromProbability(%v): hit rate %v, want ≈ %v", p, got, p)
		}
	}
}

func TestFromBits(t *testing.T) {
	// 前 bits 位全零 ⇔ 阈值 = 2^(64-bits) - 1。
	for _, tc := range []struct {
		bits int
		want uint64
	}{
		{0, math.MaxUint64}, // 无门槛，恒命中
		{1, 1<<63 - 1},      // 仅最高位为 0
		{4, 1<<60 - 1},      // 原 xbits TargetBits=4 的等价阈值
		{63, 1},             // 高 63 位全零，仅剩 val ∈ {0, 1}
	} {
		th, err := FromBits(tc.bits)
		if err != nil || uint64(th) != tc.want {
			t.Errorf("FromBits(%d) = %d, %v; want %d, nil", tc.bits, th, err, tc.want)
		}
	}

	// 越界必须报错；bits=64 时阈值为 0（实际不可达），一并拒绝。
	for _, bits := range []int{-1, 64, 100} {
		if _, err := FromBits(bits); err == nil {
			t.Errorf("FromBits(%d) should fail", bits)
		}
	}
}

func TestThresholdHit(t *testing.T) {
	th, err := FromBits(4) // 2^60 - 1
	if err != nil {
		t.Fatal(err)
	}

	mkHash := func(v uint64) *[32]byte {
		var h [32]byte
		binary.BigEndian.PutUint64(h[:8], v)
		return &h
	}

	// 边界语义：等于阈值命中，超出一位即不命中。
	if !th.hit(mkHash(uint64(th))) {
		t.Error("val == threshold should hit")
	}
	if th.hit(mkHash(uint64(th) + 1)) {
		t.Error("val == threshold+1 should miss")
	}
}

func TestSolutionBinaryRoundTrip(t *testing.T) {
	sol := &Solution{
		Nonce:    0x0102030405060708,
		Solution: equix.Solution{1, 2, 3, 0x1234, 0xffff, 7, 8, 9},
	}

	b, err := sol.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 24 {
		t.Fatalf("encoded length = %d, want 24", len(b))
	}

	var back Solution
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back != *sol {
		t.Fatalf("round trip mismatch: got %+v, want %+v", back, *sol)
	}

	// 长度不为 24 必须报错。
	for _, n := range []int{0, 8, 23, 25, 32} {
		var s Solution
		if err := s.UnmarshalBinary(make([]byte, n)); err == nil {
			t.Errorf("UnmarshalBinary(%d bytes) should fail", n)
		}
	}
}

func TestSolveAndVerify(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle test")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}

	sol, err := Solve(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if sol == nil {
		t.Fatal("Solve returned nil solution")
	}
	if !Verify(challenge, th, sol) {
		t.Fatal("Verify rejected a freshly solved puzzle")
	}

	// 错误 nonce 必须失败。
	bad := *sol
	bad.Nonce++
	if Verify(challenge, th, &bad) {
		t.Fatal("Verify accepted a mutated nonce")
	}

	// 篡改解内容必须失败。
	tampered := *sol
	tampered.Solution[0] ^= 1
	if Verify(challenge, th, &tampered) {
		t.Fatal("Verify accepted a tampered solution")
	}

	// 错误 challenge 必须失败。
	if Verify([]byte("equix-cgo/puzzle test!"), th, sol) {
		t.Fatal("Verify accepted a mismatched challenge")
	}

	// nil 提交必须失败而非 panic。
	if Verify(challenge, th, nil) {
		t.Fatal("Verify accepted a nil solution")
	}
}

func TestSolveWithHashes(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle hashes")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}

	sol, h, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}
	if sol == nil {
		t.Fatal("SolveWithHashes returned nil solution")
	}
	if !Verify(challenge, th, sol) {
		t.Fatal("Verify rejected a freshly solved puzzle")
	}
	if err := equix.VerifyHashes(h); err != nil {
		t.Fatalf("VerifyHashes: %v", err)
	}
	want, err := equix.VerifyWithHashesAndNonce(challenge, sol.Nonce, sol.Solution)
	if err != nil {
		t.Fatalf("VerifyWithHashesAndNonce: %v", err)
	}
	if h != want {
		t.Fatalf("hashes mismatch: got %v want %v", h, want)
	}
}

func TestSolveWithHashesMatchesSolve(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle hashes match")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}

	want, err := Solve(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	got, _, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}
	if got == nil || want == nil {
		t.Fatal("expected non-nil solutions")
	}
	if *got != *want {
		t.Fatalf("SolveWithHashes solution %+v != Solve %+v", *got, *want)
	}
}

func TestSolveContextWithHashesCanceled(t *testing.T) {
	th, err := FromBits(20)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sol, h, err := SolveContextWithHashes(ctx, []byte("equix-cgo/puzzle hashes cancel"), th, nonceStart)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if sol != nil {
		t.Fatalf("solution = %+v, want nil", sol)
	}
	if h != (equix.Hashes{}) {
		t.Fatalf("hashes = %v, want zero", h)
	}
}

func TestVerifyWithHashes(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle verify hashes")
	th, err := FromProbability(0.1)
	if err != nil {
		t.Fatal(err)
	}

	sol, want, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}

	got, ok := VerifyWithHashes(challenge, th, sol)
	if !ok {
		t.Fatal("VerifyWithHashes rejected a freshly solved puzzle")
	}
	if got != want {
		t.Fatalf("hashes mismatch: got %v want %v", got, want)
	}

	// 错误 nonce / 篡改解 / 错误 challenge 必须失败。
	badNonce := *sol
	badNonce.Nonce++
	if _, ok := VerifyWithHashes(challenge, th, &badNonce); ok {
		t.Fatal("VerifyWithHashes accepted a mutated nonce")
	}
	tampered := *sol
	tampered.Solution[0] ^= 1
	if _, ok := VerifyWithHashes(challenge, th, &tampered); ok {
		t.Fatal("VerifyWithHashes accepted a tampered solution")
	}
	if _, ok := VerifyWithHashes([]byte("equix-cgo/puzzle verify hashes!"), th, sol); ok {
		t.Fatal("VerifyWithHashes accepted a mismatched challenge")
	}

	// nil 提交必须失败而非 panic，且哈希为零值。
	h, ok := VerifyWithHashes(challenge, th, nil)
	if ok {
		t.Fatal("VerifyWithHashes accepted a nil solution")
	}
	if h != (equix.Hashes{}) {
		t.Fatalf("nil solution hashes = %v, want zero", h)
	}
}

func TestVerifyWithHashesStillReturnsOnOrder(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle verify hashes order")
	th, err := FromBits(0)
	if err != nil {
		t.Fatal(err)
	}
	sol, _, err := SolveWithHashes(challenge, th, nonceStart)
	if err != nil {
		t.Fatalf("SolveWithHashes: %v", err)
	}

	bad := *sol
	bad.Solution[0], bad.Solution[1] = bad.Solution[1], bad.Solution[0]

	got, ok := VerifyWithHashes(challenge, th, &bad)
	if ok {
		t.Fatal("VerifyWithHashes accepted a swapped-index solution")
	}
	want, err := equix.VerifyWithHashesAndNonce(challenge, bad.Nonce, bad.Solution)
	if !errors.Is(err, equix.ErrOrder) {
		t.Fatalf("equix.VerifyWithHashesAndNonce: %v, want ErrOrder", err)
	}
	if got != want {
		t.Fatalf("hashes mismatch: got %v want %v", got, want)
	}
	if got == (equix.Hashes{}) {
		t.Fatal("hashes must still be populated")
	}
}

func TestSolveContext(t *testing.T) {
	challenge := []byte("equix-cgo/puzzle cancel")

	// 用实际不可达的难度（命中概率 2^-20，期望耗时以小时计），
	// 使取消成为求解唯一可能的出口，保证断言的确定性。
	th, err := FromBits(20)
	if err != nil {
		t.Fatal(err)
	}

	// 已取消的 ctx：进入即返回 context.Canceled。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SolveContext(ctx, challenge, th, nonceStart); !errors.Is(err, context.Canceled) {
		t.Fatalf("SolveContext with cancelled ctx: %v, want context.Canceled", err)
	}

	// 运行中到期：限时 200ms，应返回 context.DeadlineExceeded，
	// 且确实等到了超时（而非提前出错返回）。
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	start := time.Now()
	_, err = SolveContext(ctx2, challenge, th, nonceStart)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SolveContext with timeout ctx: %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed < 150*time.Millisecond {
		t.Fatalf("SolveContext returned after %s, timeout not honoured", elapsed)
	}
}

func TestConcurrent(t *testing.T) {
	th, err := FromBits(DefaultBits)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 并发 smoke 测试：包级入口依赖主包 sync.Pool，需在 -race 下无告警。
	const workers = 4
	var wg sync.WaitGroup
	for g := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			challenge := fmt.Appendf(nil, "equix-cgo/puzzle concurrent %d", g)
			sol, err := SolveContext(ctx, challenge, th, nonceStart+uint64(g))
			if err != nil {
				t.Errorf("worker %d Solve: %v", g, err)
				return
			}
			if !Verify(challenge, th, sol) {
				t.Errorf("worker %d Verify rejected its own solution", g)
			}
		}()
	}
	wg.Wait()
}

func TestSolveCost(t *testing.T) {
	const samples = 8

	th, err := FromBits(DefaultBits)
	if err != nil {
		t.Fatal(err)
	}

	solveDurations := make([]time.Duration, 0, samples)
	verifyDurations := make([]time.Duration, 0, samples)
	nonceTries := make([]uint64, 0, samples)

	for i := range samples {
		challenge := make([]byte, 16)
		binary.LittleEndian.PutUint64(challenge, uint64(i+1))
		binary.LittleEndian.PutUint64(challenge[8:], 0x7821c0de)

		start := time.Now()
		sol, err := Solve(challenge, th, nonceStart)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("sample %d Solve: %v", i, err)
		}

		vStart := time.Now()
		ok := Verify(challenge, th, sol)
		vElapsed := time.Since(vStart)
		if !ok {
			t.Fatalf("sample %d Verify failed", i)
		}

		// tries 统计的是 nonce 尝试轮数；每轮平均产出约 1.7 个候选解。
		tries := (sol.Nonce-nonceStart)/nonceStep + 1
		solveDurations = append(solveDurations, elapsed)
		verifyDurations = append(verifyDurations, vElapsed)
		nonceTries = append(nonceTries, tries)

		t.Logf("sample %d: solve=%s verify=%s nonce=%d tries=%d",
			i, elapsed, vElapsed, sol.Nonce, tries)
	}

	// DefaultBits=4 时命中期望约 2^4=16 个候选解，折合约 16/1.7≈9 轮 nonce。
	t.Logf("solve  min=%s avg=%s max=%s",
		minDuration(solveDurations), avgDuration(solveDurations), maxDuration(solveDurations))
	t.Logf("verify min=%s avg=%s max=%s",
		minDuration(verifyDurations), avgDuration(verifyDurations), maxDuration(verifyDurations))
	t.Logf("tries  min=%d avg=%.1f max=%d (DefaultBits=%d, expected≈%.1f nonce rounds)",
		minUint64(nonceTries), avgUint64(nonceTries), maxUint64(nonceTries),
		DefaultBits, float64(1<<DefaultBits)/1.7)
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
	th, err := FromBits(DefaultBits)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; b.Loop(); i++ {
		challenge := fmt.Appendf(nil, "bench-%d", i)
		if _, err := Solve(challenge, th, nonceStart); err != nil {
			b.Fatal(err)
		}
	}
}
