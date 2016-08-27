package cid

import (
	"encoding/binary"
	mh "github.com/jbenet/go-multihash"
	mbase "github.com/multiformats/go-multibase"
)

type Cid struct {
	Version uint64
	Type    uint64
	Hash    mh.Multihash
}

func Decode(v string) (*Cid, error) {
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

func (c *Cid) Bytes() []byte {
	buf := make([]byte, 8+len(c.Hash))
	n := binary.PutUvarint(buf, c.Version)
	n += binary.PutUvarint(buf[n:], c.Type)
	copy(buf[n:], c.Hash)

	return buf[:n+len(c.Hash)]
}
