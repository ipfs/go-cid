package cid

import (
	"bytes"
	"encoding/binary"
	"fmt"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

const UnsupportedVersionString = "<unsupported cid version>"

const (
	Protobuf = 0x70
	CBOR     = 0x71
	Raw      = 0x72
	JSON     = 0x73

	EthereumBlock = 0x90
	EthereumTx    = 0x91
	BitcoinBlock  = 0xb0
	BitcoinTx     = 0xb1
	ZcashBlock    = 0xc0
	ZcashTx       = 0xc1
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
	if len(v) < 2 {
		return nil, fmt.Errorf("cid too short")
	}

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

func (c *Cid) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return fmt.Errorf("invalid cid json blob")
	}
	out, err := Decode(string(b[1 : len(b)-1]))
	if err != nil {
		return err
	}

	c.version = out.version
	c.hash = out.hash
	c.codec = out.codec
	return nil
}

func (c *Cid) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", c.String())), nil
}

func (c *Cid) KeyString() string {
	return string(c.Bytes())
}

func (c *Cid) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"cid": c,
	}
}

func (c *Cid) Prefix() Prefix {
	dec, _ := mh.Decode(c.hash) // assuming we got a valid multiaddr, this will not error
	return Prefix{
		MhType:   dec.Code,
		MhLength: dec.Length,
		Version:  c.version,
		Codec:    c.codec,
	}
}

// Prefix represents all the metadata of a cid, minus any actual content information
type Prefix struct {
	Version  uint64
	Codec    uint64
	MhType   int
	MhLength int
}

func (p Prefix) Sum(data []byte) (*Cid, error) {
	hash, err := mh.Sum(data, p.MhType, p.MhLength)
	if err != nil {
		return nil, err
	}

	switch p.Version {
	case 0:
		return NewCidV0(hash), nil
	case 1:
		return NewCidV1(p.Codec, hash), nil
	default:
		return nil, fmt.Errorf("invalid cid version")
	}
}

func (p Prefix) Bytes() []byte {
	buf := make([]byte, 16)
	n := binary.PutUvarint(buf, p.Version)
	n += binary.PutUvarint(buf[n:], p.Codec)
	n += binary.PutUvarint(buf[n:], uint64(p.MhType))
	n += binary.PutUvarint(buf[n:], uint64(p.MhLength))
	return buf[:n]
}

func PrefixFromBytes(buf []byte) (Prefix, error) {
	r := bytes.NewReader(buf)
	vers, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	codec, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	mhtype, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	mhlen, err := binary.ReadUvarint(r)
	if err != nil {
		return Prefix{}, err
	}

	return Prefix{
		Version:  vers,
		Codec:    codec,
		MhType:   int(mhtype),
		MhLength: int(mhlen),
	}, nil
}
