package equix

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestVerifyHashesAllZero(t *testing.T) {
	if err := VerifyHashes(Hashes{}); err != nil {
		t.Fatalf("zero hashes: %v", err)
	}
}

func TestVerifyHashesStage1(t *testing.T) {
	err := VerifyHashes(Hashes{1, 0, 0, 0, 0, 0, 0, 0})
	if !errors.Is(err, ErrPartialSum) {
		t.Fatalf("got %v, want ErrPartialSum", err)
	}
}

func TestVerifyHashesDoesNotReturnOrder(t *testing.T) {
	err := VerifyHashes(Hashes{1, 0, 0, 0, 0, 0, 0, 0})
	if errors.Is(err, ErrOrder) {
		t.Fatal("VerifyHashes must not return ErrOrder")
	}
}

func TestSolutionBinaryRoundTrip(t *testing.T) {
	var s Solution
	s[0], s[7] = 0x6c31, 0xe259
	b, err := s.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 16 {
		t.Fatalf("len=%d", len(b))
	}
	if binary.LittleEndian.Uint16(b[0:2]) != 0x6c31 {
		t.Fatalf("idx0 encoding")
	}
	var s2 Solution
	if err := s2.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if s2 != s {
		t.Fatalf("%v != %v", s2, s)
	}
}

func TestSolutionUnmarshalRejectsLength(t *testing.T) {
	var s Solution
	if err := s.UnmarshalBinary([]byte{1, 2, 3}); err == nil {
		t.Fatal("want error")
	}
}

func TestAppendNonce(t *testing.T) {
	got := appendNonce([]byte{0xaa}, 0x0102030405060708)
	want := []byte{0xaa, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("%x != %x", got, want)
	}
	if !bytes.Equal(appendNonce(nil, 0), make([]byte, 8)) {
		t.Fatal("nil challenge")
	}
}
