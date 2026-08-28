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
