package equix

import "errors"

var (
	ErrNotSupported = errors.New("equix: requested context type not supported")
	ErrChallenge    = errors.New("equix: invalid challenge")
	ErrOrder        = errors.New("equix: indices not in required order")
	ErrPartialSum   = errors.New("equix: partial sum missing trailing zeros")
	ErrFinalSum     = errors.New("equix: hashes do not sum to zero")
	ErrClosed       = errors.New("equix: use of closed solver or verifier")
)
