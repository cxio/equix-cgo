package equix

import "sync"

var (
	solverPool = sync.Pool{New: func() any {
		s, err := NewSolver()
		if err != nil {
			return nil
		}
		return s
	}}
	verifierPool = sync.Pool{New: func() any {
		v, err := NewVerifier()
		if err != nil {
			return nil
		}
		return v
	}}
)

func getSolver() (*Solver, error) {
	if v := solverPool.Get(); v != nil {
		return v.(*Solver), nil
	}
	return NewSolver()
}

func getVerifier() (*Verifier, error) {
	if v := verifierPool.Get(); v != nil {
		return v.(*Verifier), nil
	}
	return NewVerifier()
}

// Solve 是包级求解入口，内部通过 sync.Pool 复用 Solver，
// 可从多个 goroutine 并发调用。等价于 NewSolver 后调用其 Solve。
func Solve(challenge []byte) ([]Solution, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.Solve(challenge)
}

// Verify 是包级校验入口，内部通过 sync.Pool 复用 Verifier，
// 可从多个 goroutine 并发调用。
func Verify(challenge []byte, sol Solution) error {
	v, err := getVerifier()
	if err != nil {
		return err
	}
	defer verifierPool.Put(v)
	return v.Verify(challenge, sol)
}

// SolveWithNonce 是 Solve 的带 nonce 变体，见 Solver.SolveWithNonce。
func SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithNonce(challenge, nonce)
}

// VerifyWithNonce 是 Verify 的带 nonce 变体，见 Verifier.VerifyWithNonce。
func VerifyWithNonce(challenge []byte, nonce uint64, sol Solution) error {
	v, err := getVerifier()
	if err != nil {
		return err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithNonce(challenge, nonce, sol)
}

// SolveWithHashes 是包级入口，见 Solver.SolveWithHashes。
func SolveWithHashes(challenge []byte) ([]Result, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithHashes(challenge)
}

// SolveWithHashesAndNonce 是包级入口，见 Solver.SolveWithHashesAndNonce。
func SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithHashesAndNonce(challenge, nonce)
}

// VerifyWithHashes 是包级入口，见 Verifier.VerifyWithHashes。
func VerifyWithHashes(challenge []byte, sol Solution) (Hashes, error) {
	v, err := getVerifier()
	if err != nil {
		return Hashes{}, err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithHashes(challenge, sol)
}

// VerifyWithHashesAndNonce 是包级入口，见 Verifier.VerifyWithHashesAndNonce。
func VerifyWithHashesAndNonce(challenge []byte, nonce uint64, sol Solution) (Hashes, error) {
	v, err := getVerifier()
	if err != nil {
		return Hashes{}, err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithHashesAndNonce(challenge, nonce, sol)
}
