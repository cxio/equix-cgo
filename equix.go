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

func Solve(challenge []byte) ([]Solution, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.Solve(challenge)
}

func Verify(challenge []byte, sol Solution) error {
	v, err := getVerifier()
	if err != nil {
		return err
	}
	defer verifierPool.Put(v)
	return v.Verify(challenge, sol)
}

func SolveWithNonce(challenge []byte, nonce uint64) ([]Solution, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithNonce(challenge, nonce)
}

func VerifyWithNonce(challenge []byte, nonce uint64, sol Solution) error {
	v, err := getVerifier()
	if err != nil {
		return err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithNonce(challenge, nonce, sol)
}

func SolveWithHashes(challenge []byte) ([]Result, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithHashes(challenge)
}

func SolveWithHashesAndNonce(challenge []byte, nonce uint64) ([]Result, error) {
	s, err := getSolver()
	if err != nil {
		return nil, err
	}
	defer solverPool.Put(s)
	return s.SolveWithHashesAndNonce(challenge, nonce)
}

func VerifyWithHashes(challenge []byte, sol Solution) (Hashes, error) {
	v, err := getVerifier()
	if err != nil {
		return Hashes{}, err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithHashes(challenge, sol)
}

func VerifyWithHashesAndNonce(challenge []byte, nonce uint64, sol Solution) (Hashes, error) {
	v, err := getVerifier()
	if err != nil {
		return Hashes{}, err
	}
	defer verifierPool.Put(v)
	return v.VerifyWithHashesAndNonce(challenge, nonce, sol)
}
