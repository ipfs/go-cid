package cid

import (
	"fmt"

	mh "github.com/multiformats/go-multihash"
)

// Prefix represents all the metadata of a Cid,
// that is, the Version, the Codec, the Multihash type
// and the Multihash length. It does not contains
// any actual content information.
type Prefix struct {
	Version  uint64
	Codec    uint64
	MhType   uint64
	MhLength int
}

// Sum uses the information in a prefix to perform a multihash.Sum()
// and return a newly constructed Cid with the resulting multihash.
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

var v0CidPrefix = Prefix{
	Codec:    DagProtobuf,
	MhLength: -1,
	MhType:   mh.SHA2_256,
	Version:  0,
}

var v1CidPrefix = Prefix{
	Codec:    DagProtobuf,
	MhLength: -1,
	MhType:   mh.SHA2_256,
	Version:  1,
}

// V0CidPrefix returns a prefix for CIDv0
func V0CidPrefix() Prefix { return v0CidPrefix }

// V1CidPrefix returns a prefix for CIDv1 with the default settings
func V1CidPrefix() Prefix { return v1CidPrefix }

// PrefixForCidVersion returns the Protobuf prefix for a given CID version
func PrefixForCidVersion(version int) (Prefix, error) {
	switch version {
	case 0:
		return v0CidPrefix, nil
	case 1:
		return v1CidPrefix, nil
	default:
		return Prefix{}, fmt.Errorf("unknown CID version: %d", version)
	}
}
