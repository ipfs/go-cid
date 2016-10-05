package cid

import (
	"bytes"
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
