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

func TestUvarintEdges(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  error
	}{
		{"ErrNotMinimal", []byte{0x01 | 0x80, 0}, varint.ErrNotMinimal},
		{"ErrOverflow", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}, varint.ErrOverflow},
		{"ErrUnderflow", []byte{0x80}, varint.ErrUnderflow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, l1, err := uvarint(string(test.input))
			if err != test.want {
				t.Fatalf("error case (%v) should return varint.%s (got: %v)", test.input, test.name, err)
			}
			if v != 0 {
				t.Fatalf("error case (%v) should return 0 value (got %d)", test.input, v)
			}
			if l1 != 0 {
				t.Fatalf("error case (%v) should return 0 length (got %d)", test.input, l1)
			}
		})
	}
}
