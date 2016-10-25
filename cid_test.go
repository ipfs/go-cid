package cid

import (
	"bytes"
	"math/rand"
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func assertEqual(t *testing.T, a, b *Cid) {
	if a.codec != b.codec {
		t.Fatal("mismatch on type")
	}

	if a.version != b.version {
		t.Fatal("mismatch on version")
	}

	if !bytes.Equal(a.hash, b.hash) {
		t.Fatal("multihash mismatch")
	}
}

func TestBasicMarshaling(t *testing.T) {
	h, err := mh.Sum([]byte("TEST"), mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := &Cid{
		codec:   7,
		version: 1,
		hash:    h,
	}

	data := cid.Bytes()

	out, err := Cast(data)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cid, out)

	s := cid.String()
	out2, err := Decode(s)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cid, out2)
}

func TestEmptyString(t *testing.T) {
	_, err := Decode("")
	if err == nil {
		t.Fatal("shouldnt be able to parse an empty cid")
	}
}

func TestV0Handling(t *testing.T) {
	old := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n"

	cid, err := Decode(old)
	if err != nil {
		t.Fatal(err)
	}

	if cid.version != 0 {
		t.Fatal("should have gotten version 0 cid")
	}

	if cid.hash.B58String() != old {
		t.Fatal("marshaling roundtrip failed")
	}

	if cid.String() != old {
		t.Fatal("marshaling roundtrip failed")
	}
}

func TestV0ErrorCases(t *testing.T) {
	badb58 := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zIII"
	_, err := Decode(badb58)
	if err == nil {
		t.Fatal("should have failed to decode that ref")
	}
}

func TestPrefixRoundtrip(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(CBOR, hash)

	pref := c.Prefix()

	c2, err := pref.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	if !c.Equals(c2) {
		t.Fatal("output didnt match original")
	}

	pb := pref.Bytes()

	pref2, err := PrefixFromBytes(pb)
	if err != nil {
		t.Fatal(err)
	}

	if pref.Version != pref2.Version || pref.Codec != pref2.Codec ||
		pref.MhType != pref2.MhType || pref.MhLength != pref2.MhLength {
		t.Fatal("input prefix didnt match output")
	}
}

func TestFuzzCid(t *testing.T) {
	buf := make([]byte, 128)
	for i := 0; i < 200; i++ {
		s := rand.Intn(128)
		rand.Read(buf[:s])
		_, _ = Cast(buf[:s])
	}
}
