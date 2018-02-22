package cid

import (
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func TestValidateCids(t *testing.T) {
	assertTrue := func(v bool) {
		t.Helper()
		if !v {
			t.Fatal("expected success")
		}
	}
	assertFalse := func(v bool) {
		t.Helper()
		if v {
			t.Fatal("expected failure")
		}
	}

	assertTrue(IsGoodHash(mh.SHA2_256))
	assertTrue(IsGoodHash(mh.BLAKE2B_MIN + 32))
	assertTrue(IsGoodHash(mh.DBL_SHA2_256))
	assertTrue(IsGoodHash(mh.KECCAK_256))
	assertTrue(IsGoodHash(mh.SHA3))

	assertFalse(IsGoodHash(mh.SHA1))
	assertFalse(IsGoodHash(mh.BLAKE2B_MIN + 5))
}
