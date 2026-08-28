package native

import "testing"

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
