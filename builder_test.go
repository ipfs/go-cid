package cid

import (
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func TestFormatV1(t *testing.T) {
	data := []byte("this is some test content")

	// Construct c1
	format := V1Builder{Codec: DagCBOR, HashFun: mh.SHA2_256}
	c1, err := format.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	// Construct c2
	hash, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCidV1(DagCBOR, hash)

	if !c1.Equals(c2) {
		t.Fatal("cids mismatch")
	}
	if c1.Prefix() != c2.Prefix() {
		t.Fatal("prefixes mismatch")
	}
}

func TestFormatV0(t *testing.T) {
	data := []byte("this is some test content")

	// Construct c1
	format := V0Builder{}
	c1, err := format.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	// Construct c2
	hash, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCidV0(hash)

	if !c1.Equals(c2) {
		t.Fatal("cids mismatch")
	}
	if c1.Prefix() != c2.Prefix() {
		t.Fatal("prefixes mismatch")
	}
}
