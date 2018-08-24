package cid

import (
	"encoding/binary"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

var _ Cid = CidStr("")

// CidStr is a representation of a Cid as a string type containing binary.
//
// Using golang's string type is preferable over byte slices even for binary
// data because golang strings are immutable, usable as map keys,
// trivially comparable with built-in equals operators, etc.
type CidStr string

// EmptyCid is a constant for a zero/uninitialized/sentinelvalue cid;
// it is declared mainly for readability in checks for sentinel values.
const EmptyCid = CidStr("")

func (c CidStr) Version() uint64 {
	bytes := []byte(c)
	v, _ := binary.Uvarint(bytes)
	return v
}

func (c CidStr) Multicodec() uint64 {
	bytes := []byte(c)
	_, n := binary.Uvarint(bytes) // skip version length
	codec, _ := binary.Uvarint(bytes[n:])
	return codec
}

func (c CidStr) Multihash() mh.Multihash {
	bytes := []byte(c)
	_, n1 := binary.Uvarint(bytes)      // skip version length
	_, n2 := binary.Uvarint(bytes[n1:]) // skip codec length
	return mh.Multihash(bytes[n1+n2:])  // return slice of remainder
}

// String returns the default string representation of a Cid.
// Currently, Base58 is used as the encoding for the multibase string.
func (c CidStr) String() string {
	switch c.Version() {
	case 0:
		return c.Multihash().B58String()
	case 1:
		mbstr, err := mbase.Encode(mbase.Base58BTC, []byte(c))
		if err != nil {
			panic("should not error with hardcoded mbase: " + err.Error())
		}
		return mbstr
	default:
		panic("not possible to reach this point")
	}
}

// Prefix builds and returns a Prefix out of a Cid.
func (c CidStr) Prefix() Prefix {
	dec, _ := mh.Decode(c.Multihash()) // assuming we got a valid multiaddr, this will not error
	return Prefix{
		MhType:   dec.Code,
		MhLength: dec.Length,
		Version:  c.Version(),
		Codec:    c.Multicodec(),
	}
}
