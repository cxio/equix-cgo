package equix

import (
	"errors"
	"testing"
)

func findSolution(t *testing.T) (challenge []byte, sol Solution) {
	t.Helper()
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for n := 0; n < 32; n++ {
		ch := []byte{byte(n)}
		sols, err := s.Solve(ch)
		if err != nil {
			t.Fatal(err)
		}
		if len(sols) > 0 {
			return ch, sols[0]
		}
	}
	t.Fatal("no Equi-X v2 solution in 32 one-byte challenges")
	return nil, Solution{}
}

func TestSolveVerify(t *testing.T) {
	ch, sol := findSolution(t)
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if err := v.Verify(ch, sol); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyOrder(t *testing.T) {
	ch, sol := findSolution(t)
	sol[0], sol[1] = sol[1], sol[0]
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	if err := v.Verify(ch, sol); !errors.Is(err, ErrOrder) {
		t.Fatalf("got %v, want ErrOrder", err)
	}
}

func TestVerifyPartialSumSwap(t *testing.T) {
	ch, sol := findSolution(t)
	sol[1], sol[2] = sol[2], sol[1]
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	err = v.Verify(ch, sol)
	if !errors.Is(err, ErrPartialSum) && !errors.Is(err, ErrOrder) {
		t.Fatalf("got %v, want ErrPartialSum or ErrOrder", err)
	}
}

func TestCloseIdempotentAndUse(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Solve(nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
	var n *Solver
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyChallengeNoPanic(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sols, err := s.Solve(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sols) > 8 {
		t.Fatalf("len=%d", len(sols))
	}
	if sols == nil {
		t.Fatal("Solve must return empty slice, not nil")
	}
}

func TestNonceRoundTrip(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	ch := []byte("cxio")
	var nonce uint64 = 7
	var sols []Solution
	for n := uint64(0); n < 32 && len(sols) == 0; n++ {
		nonce = n
		sols, err = s.SolveWithNonce(ch, nonce)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(sols) == 0 {
		t.Fatal("no solution")
	}
	if err := v.VerifyWithNonce(ch, nonce, sols[0]); err != nil {
		t.Fatal(err)
	}
	if err := v.VerifyWithNonce(ch, nonce+1, sols[0]); err == nil {
		t.Fatal("wrong nonce must fail")
	}
	if err := v.VerifyWithNonce([]byte("other"), nonce, sols[0]); err == nil {
		t.Fatal("wrong challenge must fail")
	}
}

func TestNonceConcatInvariant(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	ch := []byte{0x11, 0x22}
	var nonce uint64
	var sols []Solution
	for n := uint64(0); n < 32 && len(sols) == 0; n++ {
		nonce = n
		sols, err = s.SolveWithNonce(ch, nonce)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(sols) == 0 {
		t.Fatal("no solution")
	}
	wired := appendNonce(ch, nonce)
	if err := v.Verify(wired, sols[0]); err != nil {
		t.Fatal(err)
	}
	wiredSols, err := s.Solve(wired)
	if err != nil {
		t.Fatal(err)
	}
	if len(wiredSols) != len(sols) {
		t.Fatalf("len %d != %d", len(wiredSols), len(sols))
	}
	for i := range sols {
		if wiredSols[i] != sols[i] {
			t.Fatalf("solution %d diverged", i)
		}
	}
}

func TestSolveWithHashesMatchesSolve(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ch, _ := findSolution(t)
	sols, err := s.Solve(ch)
	if err != nil {
		t.Fatal(err)
	}
	results, err := s.SolveWithHashes(ch)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(sols) {
		t.Fatalf("len %d != %d", len(results), len(sols))
	}
	for i := range sols {
		if results[i].Solution != sols[i] {
			t.Fatalf("solution %d", i)
		}
		if err := VerifyHashes(results[i].Hashes); err != nil {
			t.Fatalf("hashes %d: %v", i, err)
		}
		sum := uint64(0)
		for _, h := range results[i].Hashes {
			sum += h
		}
		if sum&((uint64(1)<<60)-1) != 0 {
			t.Fatalf("low 60 bits of sum = %x", sum)
		}
	}
}

func TestSolveWithHashesAndNonce(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ch := []byte("h")
	var nonce uint64
	var results []Result
	for n := uint64(0); n < 32 && len(results) == 0; n++ {
		nonce = n
		results, err = s.SolveWithHashesAndNonce(ch, nonce)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(results) == 0 {
		t.Fatal("no solution")
	}
	sols, err := s.SolveWithNonce(ch, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if len(sols) != len(results) {
		t.Fatal("len")
	}
	for i := range sols {
		if sols[i] != results[i].Solution {
			t.Fatalf("sol %d", i)
		}
	}
}

func TestVerifyWithHashesMatchesSolve(t *testing.T) {
	ch, sol := findSolution(t)
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	results, err := s.SolveWithHashes(ch)
	if err != nil {
		t.Fatal(err)
	}
	var want Hashes
	for _, r := range results {
		if r.Solution == sol {
			want = r.Hashes
			break
		}
	}
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	got, err := v.VerifyWithHashes(ch, sol)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%v != %v", got, want)
	}
}

func TestVerifyWithHashesStillReturnsOnOrder(t *testing.T) {
	ch, sol := findSolution(t)
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	results, err := s.SolveWithHashes(ch)
	if err != nil {
		t.Fatal(err)
	}
	var want Hashes
	for _, r := range results {
		if r.Solution == sol {
			want = r.Hashes
			break
		}
	}
	bad := sol
	bad[0], bad[1] = bad[1], bad[0]
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	got, err := v.VerifyWithHashes(ch, bad)
	if !errors.Is(err, ErrOrder) {
		t.Fatalf("got %v, want ErrOrder", err)
	}
	if got == (Hashes{}) {
		t.Fatal("hashes must still be populated")
	}
	// swapped indices → swapped first two hash slots vs valid solution
	if got[0] != want[1] || got[1] != want[0] {
		t.Fatalf("hashes should follow the (swapped) indices: got %v want swap of %v", got, want)
	}
}

func TestPackageLevelSolveVerify(t *testing.T) {
	ch, sol := findSolution(t)
	sols, err := Solve(ch)
	if err != nil {
		t.Fatal(err)
	}
	if len(sols) == 0 {
		t.Fatal("expected solutions")
	}
	if err := Verify(ch, sol); err != nil {
		t.Fatal(err)
	}
	if _, err := SolveWithNonce([]byte("p"), 0); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyWithHashesAndNonce(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	ch := []byte("n")
	var nonce uint64
	var results []Result
	for n := uint64(0); n < 32 && len(results) == 0; n++ {
		nonce = n
		results, err = s.SolveWithHashesAndNonce(ch, nonce)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(results) == 0 {
		t.Fatal("no solution")
	}
	h, err := v.VerifyWithHashesAndNonce(ch, nonce, results[0].Solution)
	if err != nil {
		t.Fatal(err)
	}
	if h != results[0].Hashes {
		t.Fatal("hash mismatch")
	}
}

func TestVerifyTreeSwapOrder(t *testing.T) {
	ch, sol := findSolution(t)
	swapped := sol
	for i := 0; i < 4; i++ {
		swapped[i], swapped[i+4] = swapped[i+4], swapped[i]
	}
	if err := Verify(ch, swapped); !errors.Is(err, ErrOrder) {
		t.Fatalf("got %v, want ErrOrder", err)
	}
}

func TestOnlyOnePermutationValid(t *testing.T) {
	if testing.Short() {
		t.Skip("40320 verifies")
	}
	ch, sol := findSolution(t)
	idx := sol
	valid := 0
	var permute func(int)
	permute = func(start int) {
		if start == 7 {
			if Verify(ch, Solution(idx)) == nil {
				valid++
			}
			return
		}
		for i := start; i < 8; i++ {
			idx[start], idx[i] = idx[i], idx[start]
			permute(start + 1)
			idx[start], idx[i] = idx[i], idx[start]
		}
	}
	permute(0)
	if valid != 1 {
		t.Fatalf("valid permutations=%d, want 1", valid)
	}
}

func BenchmarkSolve(b *testing.B) {
	s, err := NewSolver()
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	ch := []byte("bench")

	for b.Loop() {
		if _, err := s.Solve(ch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	ch, sol := func() ([]byte, Solution) {
		s, err := NewSolver()
		if err != nil {
			b.Fatal(err)
		}
		defer s.Close()
		for n := 0; n < 32; n++ {
			c := []byte{byte(n)}
			sols, err := s.Solve(c)
			if err != nil {
				b.Fatal(err)
			}
			if len(sols) > 0 {
				return c, sols[0]
			}
		}
		b.Fatal("no solution")
		return nil, Solution{}
	}()
	v, err := NewVerifier()
	if err != nil {
		b.Fatal(err)
	}
	defer v.Close()

	for b.Loop() {
		if err := v.Verify(ch, sol); err != nil {
			b.Fatal(err)
		}
	}
}
