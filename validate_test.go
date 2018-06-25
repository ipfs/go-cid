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

	assertTrue(IsGoodHash(mh.SHA1))

	assertFalse(IsGoodHash(mh.BLAKE2B_MIN + 5))

	mhcid0 := func(code uint64, length int) *Cid {
		c := &Cid{
			version: 0,
			codec:   DagProtobuf,
		}
		mhash, err := mh.Sum([]byte{}, code, length)
		if err != nil {
			t.Fatal(err)
		}
		c.hash = mhash
		return c
	}

	mhcid1 := func(code uint64, length int) *Cid {
		c := &Cid{
			version: 1,
			codec:   DagCBOR,
		}
		mhash, err := mh.Sum([]byte{}, code, length)
		if err != nil {
			t.Fatal(err)
		}
		c.hash = mhash
		return c
	}

	cases := []struct {
		cid *Cid
		err string
	}{
		{mhcid0(mh.SHA2_256, 32), ""},
		{mhcid0(mh.SHA3_256, 32), "cidv0 accepts only SHA256 hashes of standard length"},
		{mhcid0(mh.SHA2_256, 16), "cidv0 accepts only SHA256 hashes of standard length"},
		{mhcid0(mh.MURMUR3, 4), "cidv0 accepts only SHA256 hashes of standard length"},
		{mhcid1(mh.SHA2_256, 32), ""},
		{mhcid1(mh.SHA2_256, 16), "hashes must be at least 20 bytes long"},
		{mhcid1(mh.MURMUR3, 4), "potentially insecure hash functions not allowed"},
	}

	for i, cas := range cases {
		if errString(ValidateCid(cas.cid)) != cas.err {
			t.Errorf("wrong result in case of %s (index %d). Expected: %s, got %s",
				cas.cid, i, cas.err, ValidateCid(cas.cid))
		}
	}

}

func errString(err error) string {
	if err == nil {
		return ""
	} else {
		return err.Error()
	}
}
