package cid

import (
	"fmt"
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func makeCid(i int) Cid {
	data := []byte(fmt.Sprintf("this is some test content %d", i))
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	return NewCidV1(Raw, hash)
}

func TestSetRemove(t *testing.T) {
	s := NewSet()

	c1 := makeCid(1)
	s.Add(c1)

	if !s.Has(c1) {
		t.Fatal("failed to add cid")
	}

	s.Remove(c1)
	if s.Has(c1) {
		t.Fatal("failed to remove cid")
	}

	// make sure this doesn't fail, removing a removed one
	s.Remove(c1)
}

func BenchmarkSetVisit(b *testing.B) {
	s := NewSet()

	cids := make([]Cid, b.N)
	for i := 0; i < b.N; i++ {
		cids[i] = makeCid(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Visit(cids[i])
		// twice to ensure we test the adding of an existing element
		s.Visit(cids[i])
	}
	if s.Len() != b.N {
		b.FailNow()
	}
}
