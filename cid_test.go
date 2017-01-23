package cid

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func assertEqual(t *testing.T, a, b *Cid) {
	if a.Codec != b.Codec {
		t.Fatal("mismatch on type")
	}

	if a.Version != b.Version {
		t.Fatal("mismatch on version")
	}

	if !bytes.Equal(a.MHash, b.MHash) {
		t.Fatal("multihash mismatch")
	}
}

func TestBasicMarshaling(t *testing.T) {
	h, err := mh.Sum([]byte("TEST"), mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := &Cid{
		Codec:   7,
		Version: 1,
		MHash:   h,
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

	if cid.Version != 0 {
		t.Fatal("should have gotten version 0 cid")
	}

	if cid.MHash.B58String() != old {
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
	c := NewCidV1(DagCBOR, hash)

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

func Test16BytesVarint(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(DagCBOR, hash)

	c.Codec = 1 << 63
	_ = c.Bytes()
}

func TestFuzzCid(t *testing.T) {
	buf := make([]byte, 128)
	for i := 0; i < 200; i++ {
		s := rand.Intn(128)
		rand.Read(buf[:s])
		_, _ = Cast(buf[:s])
	}
}

func TestParse(t *testing.T) {
	cid, err := Parse(123)
	if err == nil {
		t.Fatalf("expected error from Parse()")
	}
	if !strings.Contains(err.Error(), "can't parse 123 as Cid") {
		t.Fatalf("expected int error, got %s", err.Error())
	}

	theHash := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n"
	h, err := mh.FromB58String(theHash)
	if err != nil {
		t.Fatal(err)
	}

	assertions := [][]interface{}{
		[]interface{}{NewCidV0(h), theHash},
		[]interface{}{NewCidV0(h).Bytes(), theHash},
		[]interface{}{h, theHash},
		[]interface{}{theHash, theHash},
		[]interface{}{"/ipfs/" + theHash, theHash},
		[]interface{}{"https://ipfs.io/ipfs/" + theHash, theHash},
		[]interface{}{"http://localhost:8080/ipfs/" + theHash, theHash},
	}

	assert := func(arg interface{}, expected string) error {
		cid, err = Parse(arg)
		if err != nil {
			return err
		}
		if cid.Version != 0 {
			return fmt.Errorf("expected version 0, got %s", string(cid.Version))
		}
		actual := cid.Hash().B58String()
		if actual != expected {
			return fmt.Errorf("expected hash %s, got %s", expected, actual)
		}
		actual = cid.String()
		if actual != expected {
			return fmt.Errorf("expected string %s, got %s", expected, actual)
		}
		return nil
	}

	for _, args := range assertions {
		err := assert(args[0], args[1].(string))
		if err != nil {
			t.Fatal(err)
		}
	}
}
