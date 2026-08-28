package native

import (
	"runtime"
	"testing"
	"time"
)

func TestNewSolverAndClose(t *testing.T) {
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s.Close()
}

func TestNewVerifierAndClose(t *testing.T) {
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	v.Close()
}

func TestVersionPins(t *testing.T) {
	if EquixCommit != "350a85dedda1344637dac09a1de786ee63a5fb01" {
		t.Fatalf("equix pin")
	}
	if HashxCommit != "08babdf4f41b0b8991d1fa94914c7c6902de0cb6" {
		t.Fatalf("hashx pin")
	}
	if HashwxCommit != "d771cbf6cdc070755f7d137cdcf9d781af14da3f" {
		t.Fatalf("hashwx pin")
	}
}

func TestContextCleanupFreesWhenDropped(t *testing.T) {
	before := freeEquixCount.Load()
	func() {
		s, err := NewSolver()
		if err != nil {
			t.Fatal(err)
		}
		if s.ptr == nil {
			t.Fatal("nil context pointer")
		}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for freeEquixCount.Load() < before+1 {
		if time.Now().After(deadline) {
			t.Fatal("equix_free was not called after dropping an unclosed solver")
		}
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCloseStopsCleanup(t *testing.T) {
	before := freeEquixCount.Load()
	s, err := NewSolver()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if got := freeEquixCount.Load(); got != before+1 {
		t.Fatalf("after Close: free count %d, want %d", got, before+1)
	}

	s = nil
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		if got := freeEquixCount.Load(); got != before+1 {
			t.Fatalf("after Close+GC: free count %d, want %d", got, before+1)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
