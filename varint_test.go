package cid

import (
	"testing"

	"github.com/multiformats/go-varint"
)

func TestUvarintRoundTrip(t *testing.T) {
	testCases := []uint64{0, 1, 2, 127, 128, 129, 255, 256, 257, 1<<63 - 1}
	for _, tc := range testCases {
		t.Log("testing", tc)
		buf := make([]byte, 16)
		varint.PutUvarint(buf, tc)
		v, l1, err := uvarint(string(buf))
		if err != nil {
			t.Fatalf("%v: %s", buf, err)
		}
		_, l2, err := varint.FromUvarint(buf)
		if err != nil {
			t.Fatal(err)
		}
		if tc != v {
			t.Errorf("roundtrip failed expected %d but got %d", tc, v)
		}
		if l1 != l2 {
			t.Errorf("length incorrect expected %d but got %d", l2, l1)
		}
	}
}
