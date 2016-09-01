package cid

import (
	"bytes"
	"encoding/binary"
	"fmt"

	mbase "github.com/multiformats/go-multibase"
	mh "gx/ipfs/QmYf7ng2hG5XBtJA3tN34DQ2GUN5HNksEw1rLDkmr6vGku/go-multihash"
)

const UnsupportedVersionString = "<unsupported cid version>"

const (
	Protobuf = iota
	Raw
	JSON
	CBOR
)

func NewCidV0(h mh.Multihash) *Cid {
	return &Cid{
		version: 0,
		codec:   Protobuf,
		hash:    h,
	}
}

func NewCidV1(c uint64, h mh.Multihash) *Cid {
	return &Cid{
		version: 1,
		codec:   c,
		hash:    h,
	}
}

type Cid struct {
	version uint64
	codec   uint64
	hash    mh.Multihash
}

func Decode(v string) (*Cid, error) {
	if len(v) == 46 && v[:2] == "Qm" {
		hash, err := mh.FromB58String(v)
		if err != nil {
			return nil, err
		}

		return NewCidV0(hash), nil
	}

	_, data, err := mbase.Decode(v)
	if err != nil {
		return nil, err
	}

	return Cast(data)
}

func Cast(data []byte) (*Cid, error) {
	if len(data) == 34 && data[0] == 18 && data[1] == 32 {
		h, err := mh.Cast(data)
		if err != nil {
			return nil, err
		}

		return &Cid{
			codec:   Protobuf,
			version: 0,
			hash:    h,
		}, nil
	}

	vers, n := binary.Uvarint(data)
	if vers != 0 && vers != 1 {
		return nil, fmt.Errorf("invalid cid version number: %d", vers)
	}

	codec, cn := binary.Uvarint(data[n:])

	rest := data[n+cn:]
	h, err := mh.Cast(rest)
	if err != nil {
		return nil, err
	}

	return &Cid{
		version: vers,
		codec:   codec,
		hash:    h,
	}, nil
}

func (c *Cid) Type() uint64 {
	return c.codec
}

func (c *Cid) String() string {
	switch c.version {
	case 0:
		return c.hash.B58String()
	case 1:
		mbstr, err := mbase.Encode(mbase.Base58BTC, c.bytesV1())
		if err != nil {
			panic("should not error with hardcoded mbase: " + err.Error())
		}

		return mbstr
	default:
		panic("not possible to reach this point")
	}
}

func (c *Cid) Hash() mh.Multihash {
	return c.hash
}

func (c *Cid) Bytes() []byte {
	switch c.version {
	case 0:
		return c.bytesV0()
	case 1:
		return c.bytesV1()
	default:
		panic("not possible to reach this point")
	}
}

func (c *Cid) bytesV0() []byte {
	return []byte(c.hash)
}

func (c *Cid) bytesV1() []byte {
	buf := make([]byte, 8+len(c.hash))
	n := binary.PutUvarint(buf, c.version)
	n += binary.PutUvarint(buf[n:], c.codec)
	copy(buf[n:], c.hash)

	return buf[:n+len(c.hash)]
}

func (c *Cid) Equals(o *Cid) bool {
	return c.codec == o.codec &&
		c.version == o.version &&
		bytes.Equal(c.hash, o.hash)
}
