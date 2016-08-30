package cid

import (
	"encoding/binary"
	"fmt"

	mh "github.com/jbenet/go-multihash"
	mbase "github.com/multiformats/go-multibase"
)

const UnsupportedVersionString = "<unsupported cid version>"

type Cid struct {
	Version uint64
	Type    uint64
	Hash    mh.Multihash
}

func Decode(v string) (*Cid, error) {
	if len(v) == 46 && v[:2] == "Qm" {
		hash, err := mh.FromB58String(v)
		if err != nil {
			return nil, err
		}

		return &Cid{
			Version: 0,
			Hash:    hash,
		}, nil
	}

	_, data, err := mbase.Decode(v)
	if err != nil {
		return nil, err
	}

	return Cast(data)
}

func Cast(data []byte) (*Cid, error) {
	vers, n := binary.Uvarint(data)
	codec, cn := binary.Uvarint(data[n:])

	rest := data[n+cn:]
	h, err := mh.Cast(rest)
	if err != nil {
		return nil, err
	}

	return &Cid{
		Version: vers,
		Type:    codec,
		Hash:    h,
	}, nil
}

func (c *Cid) String() string {
	switch c.Version {
	case 0:
		return c.Hash.B58String()
	case 1:
		mbstr, err := mbase.Encode(mbase.Base58BTC, c.bytesV1())
		if err != nil {
			panic("should not error with hardcoded mbase: " + err.Error())
		}

		return mbstr
	default:
		return "<unsupported cid version>"
	}
}

func (c *Cid) Bytes() ([]byte, error) {
	switch c.Version {
	case 0:
		return c.bytesV0(), nil
	case 1:
		return c.bytesV1(), nil
	default:
		return nil, fmt.Errorf("unsupported cid version")
	}
}

func (c *Cid) bytesV0() []byte {
	return []byte(c.Hash)
}

func (c *Cid) bytesV1() []byte {
	buf := make([]byte, 8+len(c.Hash))
	n := binary.PutUvarint(buf, c.Version)
	n += binary.PutUvarint(buf[n:], c.Type)
	copy(buf[n:], c.Hash)

	return buf[:n+len(c.Hash)]
}
