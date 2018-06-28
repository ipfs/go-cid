package cid

import (
	"fmt"

	mh "github.com/multiformats/go-multihash"
)

// TODO: Move these to the correct go-ipld-* crates?

// DagProtobufV0Format is the default CID construction for DagProtobuf (CID V0)
// nodes.
var DagProtobufV0Format = Format{
	version:   0,
	codec:     DagProtobuf,
	mhLength:  -1,
	mhType:    mh.SHA2_256,
	inlineMax: -1,
}

// DagProtobufV1Format is the default CID construction for DagProtobuf (CID V1)
// nodes.
var DagProtobufV1Format = Format{
	version:   1,
	codec:     DagProtobuf,
	mhLength:  -1,
	mhType:    mh.SHA2_256,
	inlineMax: -1,
}

// RawFormat is the default CID construction for Raw IPLD nodes.
var RawFormat = Format{
	version:   1,
	codec:     Raw,
	mhType:    mh.SHA2_256,
	mhLength:  -1,
	inlineMax: -1,
}

// DagCBORFormat is the default CID construction for CBOR IPLD nodes.
var DagCBORFormat = Format{
	version:   1,
	codec:     DagCBOR,
	mhType:    mh.SHA2_256,
	mhLength:  -1,
	inlineMax: -1,
}

// Format represents a CID format.
type Format struct {
	// CID spec
	codec   uint64
	version uint64

	// multhash spec
	mhType   uint64
	mhLength int

	// inline when the data is at most inlineMax in length
	inlineMax int
}

// TODO: detect bad formats? Probably not worth it. We catch them on Sum anyways.

// With extends the format with the given options, returning a new format.
func (f Format) With(opts ...FormatOption) Format {
	for _, opt := range opts {
		opt(&f)
	}
	return f
}

// Sum constructs a CID for the given data. It *does not* check any properties
// of the data.
func (f *Format) Sum(data []byte) (*Cid, error) {
	mhType := f.mhType
	mhLen := f.mhLength
	if len(data) <= f.inlineMax {
		mhType = mh.ID
		mhLen = -1
	}

	hash, err := mh.Sum(data, mhType, mhLen)
	if err != nil {
		return nil, err
	}

	switch f.version {
	case 0:
		if f.inlineMax != 0 {
			return nil, fmt.Errorf("cannot inline with V0 CIDs")
		}
		if f.mhType != mh.SHA2_256 || (f.mhLength != -1 && f.mhLength != 256) {
			return nil, fmt.Errorf("CIDv0 only supports 256bit SHA2_256 hashes")
		}
		return NewCidV0(hash), nil
	case 1:
		return NewCidV1(f.codec, hash), nil
	default:
		return nil, fmt.Errorf("invalid cid version")
	}
}

// FormatOption is a format option.
type FormatOption func(f *Format)

// OptCodec configures the format to use the specified IPLD codec.
func OptCodec(codec uint64) FormatOption {
	return func(f *Format) {
		f.codec = codec
	}
}

// OptInline configures the format to auto-inline (using the identity hash
// function) objects at most maxSize in length.
//
// Specify a value < 0 to disable auto-inlining.
func OptInline(maxSize int) FormatOption {
	return func(f *Format) {
		f.inlineMax = maxSize
	}
}

// TODO: Split length and hash?

// OptHash sets the hash function to use in the format. Pass -1 as the length to
// use the default.
func OptHash(hash uint64, length int) FormatOption {
	return func(f *Format) {
		f.mhType = hash
		f.mhLength = length
	}
}

// TODO: Do we actually want this or should users always extend Format
// That is, DagCBORFormat.With(OptHash(...))

// NewFormat constructs a new CID format. It defaults to:
//
// * The DagCBOR format.
// * The SHA256 hash function.
func NewFormat(opts ...FormatOption) Format {
	return DagCBORFormat.With(opts...)
}
