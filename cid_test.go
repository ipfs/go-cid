package cid

import (
	"bytes"
	"testing"

	mh "github.com/jbenet/go-multihash"
)

func TestBasicMarshaling(t *testing.T) {
	h, err := mh.Sum([]byte("TEST"), mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := &Cid{
		Type:    7,
		Version: 1,
		Hash:    h,
	}

	data := cid.Bytes()

	out, err := Cast(data)
	if err != nil {
		t.Fatal(err)
	}

	if out.Type != cid.Type {
		t.Fatal("mismatch on type")
	}

	if out.Version != cid.Version {
		t.Fatal("mismatch on version")
	}

	if !bytes.Equal(out.Hash, cid.Hash) {
		t.Fatal("multihash mismatch")
	}
}
