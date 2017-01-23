package cid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

const UnsupportedVersionString = "<unsupported cid version>"

const (
	Raw = 0x55

	DagProtobuf = 0x70
	DagCBOR     = 0x71

	EthereumBlock = 0x90
	EthereumTx    = 0x91
	BitcoinBlock  = 0xb0
	BitcoinTx     = 0xb1
	ZcashBlock    = 0xc0
	ZcashTx       = 0xc1
)

func NewCidV0(h mh.Multihash) *Cid {
	return &Cid{
		Version: 0,
		Codec:   DagProtobuf,
		MHash:   h,
	}
}

func NewCidV1(c uint64, h mh.Multihash) *Cid {
	return &Cid{
		Version: 1,
		Codec:   c,
		MHash:   h,
	}
}

type Cid struct {
	Version uint64
	Codec   uint64
	MHash   mh.Multihash
}

func Parse(v interface{}) (*Cid, error) {
	switch v2 := v.(type) {
	case string:
		if strings.Contains(v2, "/ipfs/") {
			return Decode(strings.Split(v2, "/ipfs/")[1])
		}
		return Decode(v2)
	case []byte:
		return Cast(v2)
	case mh.Multihash:
		return NewCidV0(v2), nil
	case *Cid:
		return v2, nil
	default:
		return nil, fmt.Errorf("can't parse %+v as Cid", v2)
	}
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

var (
	ErrVarintBuffSmall = errors.New("reading varint: buffer to small")
	ErrVarintTooBig    = errors.New("reading varint: varint bigger than 64bits" +
		" and not supported")
)

func uvError(read int) error {
	switch {
	case read == 0:
		return ErrVarintBuffSmall
	case read < 0:
		return ErrVarintTooBig
	default:
		return nil
	}
}

func Cast(data []byte) (*Cid, error) {
	if len(data) == 34 && data[0] == 18 && data[1] == 32 {
		h, err := mh.Cast(data)
		if err != nil {
			return nil, err
		}

		return &Cid{
			Codec:   DagProtobuf,
			Version: 0,
			MHash:   h,
		}, nil
	}

	vers, n := binary.Uvarint(data)
	if err := uvError(n); err != nil {
		return nil, err
	}

	if vers != 0 && vers != 1 {
		return nil, fmt.Errorf("invalid cid version number: %d", vers)
	}

	codec, cn := binary.Uvarint(data[n:])
	if err := uvError(cn); err != nil {
		return nil, err
	}

	rest := data[n+cn:]
	h, err := mh.Cast(rest)
	if err != nil {
		return nil, err
	}

	return &Cid{
		Version: vers,
		Codec:   codec,
		MHash:   h,
	}, nil
}

func (c *Cid) Type() uint64 {
	return c.Codec
}

func (c *Cid) String() string {
	switch c.Version {
	case 0:
		return c.MHash.B58String()
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
	return c.MHash
}

func (c *Cid) Bytes() []byte {
	switch c.Version {
	case 0:
		return c.bytesV0()
	case 1:
		return c.bytesV1()
	default:
		panic("not possible to reach this point")
	}
}

func (c *Cid) bytesV0() []byte {
	return []byte(c.MHash)
}

func (c *Cid) bytesV1() []byte {
	// two 8 bytes (max) numbers plus hash
	buf := make([]byte, 2*binary.MaxVarintLen64+len(c.MHash))
	n := binary.PutUvarint(buf, c.Version)
	n += binary.PutUvarint(buf[n:], c.Codec)
	cn := copy(buf[n:], c.MHash)
	if cn != len(c.MHash) {
		panic("copy hash length is inconsistent")
	}

	return buf[:n+len(c.MHash)]
}

func (c *Cid) Equals(o *Cid) bool {
	return c.Codec == o.Codec &&
		c.Version == o.Version &&
		bytes.Equal(c.MHash, o.MHash)
}

func (c *Cid) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return fmt.Errorf("invalid cid json blob")
	}
	out, err := Decode(string(b[1 : len(b)-1]))
	if err != nil {
		return err
	}

	c.Version = out.Version
	c.MHash = out.MHash
	c.Codec = out.Codec
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
	dec, _ := mh.Decode(c.MHash) // assuming we got a valid multiaddr, this will not error
	return Prefix{
		MhType:   int(dec.Code),
		MhLength: dec.Length,
		Version:  c.Version,
		Codec:    c.Codec,
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
	buf := make([]byte, 4*binary.MaxVarintLen64)
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
