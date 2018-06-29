package cid

import (
	mh "github.com/multiformats/go-multihash"
)

type Format interface {
	Sum(data []byte) (*Cid, error)
}

type FormatV0 struct{}

type FormatV1 struct {
	Codec   uint64
	HashFun uint64
	HashLen int // HashLen <= 0 means the default length
}

func PrefixToFormat(p Prefix) Format {
	if p.Version == 0 {
		return FormatV0{}
	}
	mhLen := p.MhLength
	if p.MhType == mh.ID {
		mhLen = 0
	}
	if mhLen < 0 {
		mhLen = 0
	}
	return FormatV1{
		Codec:   p.Codec,
		HashFun: p.MhType,
		HashLen: mhLen,
	}
}

func (p FormatV0) Sum(data []byte) (*Cid, error) {
	hash, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		return nil, err
	}
	return NewCidV0(hash), nil
}

func (p FormatV1) Sum(data []byte) (*Cid, error) {
	mhLen := p.HashLen
	if mhLen <= 0 {
		mhLen = -1
	}
	hash, err := mh.Sum(data, p.HashFun, mhLen)
	if err != nil {
		return nil, err
	}
	return NewCidV1(p.Codec, hash), nil
}
