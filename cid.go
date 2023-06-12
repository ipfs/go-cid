// Package cid implements the Content-IDentifiers specification
// (https://github.com/ipld/cid) in Go. CIDs are
// self-describing content-addressed identifiers useful for
// distributed information systems. CIDs are used in the IPFS
// (https://ipfs.io) project ecosystem.
//
// CIDs have two major versions. A CIDv0 corresponds to a multihash of type
// DagProtobuf, is deprecated and exists for compatibility reasons. Usually,
// CIDv1 should be used.
//
// A CIDv1 has four parts:
//
//	<cidv1> ::= <multibase-prefix><cid-version><multicodec-packed-content-type><multihash-content-address>
//
// As shown above, the CID implementation relies heavily on Multiformats,
// particularly Multibase
// (https://github.com/multiformats/go-multibase), Multicodec
// (https://github.com/multiformats/multicodec) and Multihash
// implementations (https://github.com/multiformats/go-multihash).
package cid

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
	varint "github.com/multiformats/go-varint"
)

// UnsupportedVersionString just holds an error message
const UnsupportedVersionString = "<unsupported cid version>"

// ErrInvalidCid is an error that indicates that a CID is invalid.
type ErrInvalidCid struct {
	Err error
}

func (e ErrInvalidCid) Error() string {
	return fmt.Sprintf("invalid cid: %s", e.Err)
}

func (e ErrInvalidCid) Unwrap() error {
	return e.Err
}

func (e ErrInvalidCid) Is(err error) bool {
	switch err.(type) {
	case ErrInvalidCid, *ErrInvalidCid:
		return true
	default:
		return false
	}
}

var (
	// ErrCidTooShort means that the cid passed to decode was not long
	// enough to be a valid Cid
	ErrCidTooShort = ErrInvalidCid{errors.New("cid too short")}

	// ErrInvalidEncoding means that selected encoding is not supported
	// by this Cid version
	ErrInvalidEncoding = errors.New("invalid base encoding")
)

// Consts below are DEPRECATED and left only for legacy reasons:
// <https://github.com/ipfs/go-cid/pull/137>
// Modern code should use consts from go-multicodec instead:
// <https://github.com/multiformats/go-multicodec>
const (
	// common ones
	Raw         = 0x55
	DagProtobuf = 0x70   // https://ipld.io/docs/codecs/known/dag-pb/
	DagCBOR     = 0x71   // https://ipld.io/docs/codecs/known/dag-cbor/
	DagJSON     = 0x0129 // https://ipld.io/docs/codecs/known/dag-json/
	Libp2pKey   = 0x72   // https://github.com/libp2p/specs/blob/master/peer-ids/peer-ids.md#peer-ids

	// other
	GitRaw                = 0x78
	DagJOSE               = 0x85 // https://ipld.io/specs/codecs/dag-jose/spec/
	EthBlock              = 0x90
	EthBlockList          = 0x91
	EthTxTrie             = 0x92
	EthTx                 = 0x93
	EthTxReceiptTrie      = 0x94
	EthTxReceipt          = 0x95
	EthStateTrie          = 0x96
	EthAccountSnapshot    = 0x97
	EthStorageTrie        = 0x98
	BitcoinBlock          = 0xb0
	BitcoinTx             = 0xb1
	ZcashBlock            = 0xc0
	ZcashTx               = 0xc1
	DecredBlock           = 0xe0
	DecredTx              = 0xe1
	DashBlock             = 0xf0
	DashTx                = 0xf1
	FilCommitmentUnsealed = 0xf101
	FilCommitmentSealed   = 0xf102
)

// tryNewCidV0 tries to convert a multihash into a CIDv0 CID and returns an
// error on failure.
func tryNewCidV0[S Storage](mhash mh.Multihash) (CidOf[S], error) {
	var zero CidOf[S]
	// Need to make sure hash is valid for CidV0 otherwise we will
	// incorrectly detect it as CidV1 in the Version() method
	dec, err := mh.Decode(mhash)
	if err != nil {
		return zero, ErrInvalidCid{err}
	}
	if dec.Code != mh.SHA2_256 || dec.Length != 32 {
		return zero, ErrInvalidCid{fmt.Errorf("invalid hash for cidv0 %d-%d", dec.Code, dec.Length)}
	}
	return CidOf[S]{S(mhash)}, nil
}

// NewCidV0 returns a Cid-wrapped multihash.
// They exist to allow IPFS to work with Cids while keeping
// compatibility with the plain-multihash format used used in IPFS.
// NewCidV1 should be used preferentially.
//
// Panics if the multihash isn't sha2-256.
func NewCidV0(mhash mh.Multihash) Cid {
	return NewCidV0Of[string](mhash)
}

// NewCidV0Of is like [NewCidV0].
func NewCidV0Of[S Storage](mhash mh.Multihash) CidOf[S] {
	c, err := tryNewCidV0[S](mhash)
	if err != nil {
		panic(err)
	}
	return c
}

// NewCidV1 returns a new Cid using the given multicodec-packed
// content type.
//
// Panics if the multihash is invalid.
func NewCidV1(codecType uint64, mhash mh.Multihash) Cid {
	return NewCidV1Of[string](codecType, mhash)
}

// NewCidV1Of is like [NewCidV1].
func NewCidV1Of[S Storage](codecType uint64, mhash mh.Multihash) CidOf[S] {
	// Two 8 bytes (max) numbers plus hash.
	// We use strings.Builder to only allocate once.
	const versionSize = 1
	b := make([]byte, versionSize+varint.UvarintSize(codecType)+len(mhash))
	b[0] = 1

	n := varint.PutUvarint(b[versionSize:], codecType)

	cn := copy(b[versionSize+n:], mhash)
	if cn != len(mhash) {
		panic("copy hash length is inconsistent")
	}

	return CidOf[S]{S(b)}
}

var (
	_ encoding.BinaryMarshaler   = Cid{}
	_ encoding.BinaryUnmarshaler = (*Cid)(nil)
	_ encoding.TextMarshaler     = Cid{}
	_ encoding.TextUnmarshaler   = (*Cid)(nil)
)

// Cid represents a self-describing content addressed
// identifier. It is formed by a Version, a Codec (which indicates
// a multicodec-packed content type) and a Multihash.
type Cid = CidOf[string]

// Storage defines usable backing containers for [CidOf].
// Using [string] allows for compile-time immutable CIDs that are comparable.
// Using a byte slice allows for aliasable CIDs and zero allocation.
// When a byte slice is used it MUST NOT be modified while the cid is alive.
type Storage interface{ string | []byte }

// CidOf is like [Cid] except it's backing array is generic.
type CidOf[S Storage] struct{ storage S }

// Undef can be used to represent a nil or undefined [Cid], using Cid{}
// directly is also acceptable.
var Undef = Cid{}

// ToString is a helper that converts to a [string] backed CID.
func (c CidOf[S]) ToString() Cid {
	return Cid{string(c.storage)}
}

// ToBytes is a helper that converts to a byte slice backed CID.
func (c CidOf[S]) ToBytes() CidOf[[]byte] {
	return CidOf[[]byte]{[]byte(c.storage)}
}

// Defined returns true if a Cid is defined
// Calling any other methods on an undefined Cid will result in
// undefined behavior.
func (c CidOf[S]) Defined() bool {
	return len(c.storage) != 0
}

// Parse is a short-hand function to perform Decode, Cast etc... on
// a generic interface{} type.
func Parse(v interface{}) (Cid, error) {
	return ParseOf[string](v)
}

// ParseOf is like [Parse].
func ParseOf[S Storage](v interface{}) (CidOf[S], error) {
	switch v2 := v.(type) {
	case string:
		if strings.Contains(v2, "/ipfs/") {
			return DecodeOf[S](strings.Split(v2, "/ipfs/")[1])
		}
		return DecodeOf[S](v2)
	case []byte:
		return CastOf[S](v2)
	case mh.Multihash:
		return tryNewCidV0[S](v2)
	case CidOf[S]:
		return v2, nil
	default:
		var zero CidOf[S]
		return zero, ErrInvalidCid{fmt.Errorf("can't parse %+v as Cid", v2)}
	}
}

// MustParse calls Parse but will panic on error.
func MustParse(v interface{}) Cid {
	return MustParseOf[string](v)
}

// MustParseOf is like [MustParse].
func MustParseOf[S Storage](v interface{}) CidOf[S] {
	c, err := ParseOf[S](v)
	if err != nil {
		panic(err)
	}
	return c
}

// Decode parses a Cid-encoded string and returns a Cid object.
// For CidV1, a Cid-encoded string is primarily a multibase string:
//
//	<multibase-type-code><base-encoded-string>
//
// The base-encoded string represents a:
//
// <version><codec-type><multihash>
//
// Decode will also detect and parse CidV0 strings. Strings
// starting with "Qm" are considered CidV0 and treated directly
// as B58-encoded multihashes.
func Decode(v string) (Cid, error) {
	return DecodeOf[string](v)
}

// DecodeOf is like [Decode].
func DecodeOf[S Storage](v string) (CidOf[S], error) {
	var zero CidOf[S]
	if len(v) < 2 {
		return zero, ErrCidTooShort
	}

	if len(v) == 46 && v[:2] == "Qm" {
		hash, err := mh.FromB58String(v)
		if err != nil {
			return zero, ErrInvalidCid{err}
		}

		return tryNewCidV0[S](hash)
	}

	_, data, err := mbase.Decode(v)
	if err != nil {
		return zero, ErrInvalidCid{err}
	}

	return CastOf[S](data)
}

// Extract the encoding from a Cid.  If Decode on the same string did
// not return an error neither will this function.
func ExtractEncoding(v string) (mbase.Encoding, error) {
	if len(v) < 2 {
		return -1, ErrCidTooShort
	}

	if len(v) == 46 && v[:2] == "Qm" {
		return mbase.Base58BTC, nil
	}

	encoding := mbase.Encoding(v[0])

	// check encoding is valid
	_, err := mbase.NewEncoder(encoding)
	if err != nil {
		return -1, ErrInvalidCid{err}
	}

	return encoding, nil
}

// Cast takes a Cid data slice, parses it and returns a Cid.
// For CidV1, the data buffer is in the form:
//
//	<version><codec-type><multihash>
//
// CidV0 are also supported. In particular, data buffers starting
// with length 34 bytes, which starts with bytes [18,32...] are considered
// binary multihashes.
//
// Please use decode when parsing a regular Cid string, as Cast does not
// expect multibase-encoded data. Cast accepts the output of Cid.Bytes().
func Cast(data []byte) (Cid, error) {
	return CastOf[string](data)
}

// CastOf is like [Cast].
func CastOf[S Storage](data []byte) (CidOf[S], error) {
	var zero CidOf[S]
	nr, c, err := CidFromBytesOf[S](data)
	if err != nil {
		return zero, ErrInvalidCid{err}
	}

	if nr != len(data) {
		return zero, ErrInvalidCid{fmt.Errorf("trailing bytes in data buffer passed to cid Cast")}
	}

	return c, nil
}

// UnmarshalBinary is equivalent to Cast(). It implements the
// encoding.BinaryUnmarshaler interface.
func (c *CidOf[S]) UnmarshalBinary(data []byte) error {
	casted, err := CastOf[S](data)
	if err != nil {
		return err
	}
	c.storage = casted.storage
	return nil
}

// UnmarshalText is equivalent to Decode(). It implements the
// encoding.TextUnmarshaler interface.
func (c *CidOf[S]) UnmarshalText(text []byte) error {
	decodedCid, err := DecodeOf[S](string(text))
	if err != nil {
		return err
	}
	c.storage = decodedCid.storage
	return nil
}

// Version returns the Cid version.
func (c CidOf[S]) Version() uint64 {
	if len(c.storage) == 34 && c.storage[0] == 18 && c.storage[1] == 32 {
		return 0
	}
	return 1
}

// Type returns the multicodec-packed content type of a Cid.
func (c CidOf[S]) Type() uint64 {
	if c.Version() == 0 {
		return DagProtobuf
	}
	_, n, _ := uvarint(c.storage)
	codec, _, _ := uvarint(c.storage[n:])
	return codec
}

// String returns the default string representation of a
// Cid. Currently, Base32 is used for CIDV1 as the encoding for the
// multibase string, Base58 is used for CIDV0.
func (c CidOf[S]) String() string {
	switch c.Version() {
	case 0:
		return c.Hash().B58String()
	case 1:
		mbstr, err := mbase.Encode(mbase.Base32, c.Bytes())
		if err != nil {
			panic("should not error with hardcoded mbase: " + err.Error())
		}

		return mbstr
	default:
		panic("not possible to reach this point")
	}
}

// String returns the string representation of a Cid
// encoded is selected base
func (c CidOf[S]) StringOfBase(base mbase.Encoding) (string, error) {
	switch c.Version() {
	case 0:
		if base != mbase.Base58BTC {
			return "", ErrInvalidEncoding
		}
		return c.Hash().B58String(), nil
	case 1:
		return mbase.Encode(base, c.Bytes())
	default:
		panic("not possible to reach this point")
	}
}

// Encode return the string representation of a Cid in a given base
// when applicable.  Version 0 Cid's are always in Base58 as they do
// not take a multibase prefix.
func (c CidOf[S]) Encode(base mbase.Encoder) string {
	switch c.Version() {
	case 0:
		return c.Hash().B58String()
	case 1:
		return base.Encode(c.Bytes())
	default:
		panic("not possible to reach this point")
	}
}

// Hash returns the multihash contained by a Cid.
func (c CidOf[S]) Hash() mh.Multihash {
	bytes := c.Bytes()

	if c.Version() == 0 {
		return mh.Multihash(bytes)
	}

	// skip version length
	_, n1, _ := varint.FromUvarint(bytes)
	// skip codec length
	_, n2, _ := varint.FromUvarint(bytes[n1:])

	return mh.Multihash(bytes[n1+n2:])
}

// Bytes returns the byte representation of a Cid.
// The output of bytes can be parsed back into a Cid
// with Cast().
//
// If c.Defined() == false, it return a nil slice and may not
// be parsable with Cast().
func (c CidOf[S]) Bytes() []byte {
	if !c.Defined() {
		return nil
	}
	return []byte(c.storage)
}

// ByteLen returns the length of the CID in bytes.
// It's equivalent to `len(c.Bytes())`, but works without an allocation,
// and should therefore be preferred.
//
// (See also the WriteTo method for other important operations that work without allocation.)
func (c CidOf[S]) ByteLen() int {
	return len(c.storage)
}

// WriteBytes writes the CID bytes to the given writer.
// This method works without incurring any allocation.
//
// (See also the ByteLen method for other important operations that work without allocation.)
func (c CidOf[S]) WriteBytes(w io.Writer) (n int, err error) {
	switch v := any(c.storage).(type) {
	case string:
		n, err = io.WriteString(w, v)
	case []byte:
		n, err = w.Write(v)
	}
	if err != nil {
		return n, err
	}
	if n != len(c.storage) {
		return n, fmt.Errorf("failed to write entire cid string")
	}
	return n, nil
}

// MarshalBinary is equivalent to Bytes(). It implements the
// encoding.BinaryMarshaler interface.
func (c CidOf[S]) MarshalBinary() ([]byte, error) {
	return c.Bytes(), nil
}

// MarshalText is equivalent to String(). It implements the
// encoding.TextMarshaler interface.
func (c CidOf[S]) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// Equals checks that two Cids are the same.
// In order for two Cids to be considered equal, the
// Version, the Codec and the Multihash must match.
func (c CidOf[S]) Equals(o CidOf[S]) bool {
	return string(c.storage) == string(o.storage)
}

// EqualsString is like [Equals] but it takes in a [Cid] to compare to.
func (c CidOf[S]) EqualsString(o Cid) bool {
	return string(c.storage) == string(o.storage)
}

// EqualsBytes is like [Equals] but it takes in a [CidOf[[]byte]] to compare to.
func (c CidOf[S]) EqualsBytes(o CidOf[[]byte]) bool {
	return string(c.storage) == string(o.storage)
}

// Equals is like [CidOf.Equals] but it is a function to allow to compare two different cids
// as go doesn't support generic methods.
func Equals[A, B Storage](a CidOf[A], b CidOf[B]) bool {
	return string(a.storage) == string(b.storage)
}

// UnmarshalJSON parses the JSON representation of a Cid.
func (c *CidOf[S]) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return ErrInvalidCid{fmt.Errorf("invalid cid json blob")}
	}
	obj := struct {
		CidTarget string `json:"/"`
	}{}
	objptr := &obj
	err := json.Unmarshal(b, &objptr)
	if err != nil {
		return ErrInvalidCid{err}
	}
	if objptr == nil {
		*c = CidOf[S]{}
		return nil
	}

	if obj.CidTarget == "" {
		return ErrInvalidCid{fmt.Errorf("cid was incorrectly formatted")}
	}

	out, err := DecodeOf[S](obj.CidTarget)
	if err != nil {
		return ErrInvalidCid{err}
	}

	*c = out

	return nil
}

// MarshalJSON procudes a JSON representation of a Cid, which looks as follows:
//
//	{ "/": "<cid-string>" }
//
// Note that this formatting comes from the IPLD specification
// (https://github.com/ipld/specs/tree/master/ipld)
func (c CidOf[S]) MarshalJSON() ([]byte, error) {
	if !c.Defined() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("{\"/\":\"%s\"}", c.String())), nil
}

// KeyString returns the binary representation of the Cid as a string
func (c CidOf[S]) KeyString() string {
	return string(c.storage)
}

// Loggable returns a Loggable (as defined by
// https://godoc.org/github.com/ipfs/go-log).
func (c CidOf[S]) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"cid": c,
	}
}

// Prefix builds and returns a Prefix out of a Cid.
func (c CidOf[S]) Prefix() Prefix {
	if c.Version() == 0 {
		return Prefix{
			MhType:   mh.SHA2_256,
			MhLength: 32,
			Version:  0,
			Codec:    DagProtobuf,
		}
	}

	offset := 0
	version, n, _ := uvarint(c.storage[offset:])
	offset += n
	codec, n, _ := uvarint(c.storage[offset:])
	offset += n
	mhtype, n, _ := uvarint(c.storage[offset:])
	offset += n
	mhlen, _, _ := uvarint(c.storage[offset:])

	return Prefix{
		MhType:   mhtype,
		MhLength: int(mhlen),
		Version:  version,
		Codec:    codec,
	}
}

// Prefix represents all the metadata of a Cid,
// that is, the Version, the Codec, the Multihash type
// and the Multihash length. It does not contains
// any actual content information.
// NOTE: The use -1 in MhLength to mean default length is deprecated,
//
//	use the V0Builder or V1Builder structures instead
type Prefix struct {
	Version  uint64
	Codec    uint64
	MhType   uint64
	MhLength int
}

// Sum uses the information in a prefix to perform a multihash.Sum()
// and return a newly constructed Cid with the resulting multihash.
func (p Prefix) Sum(data []byte) (Cid, error) {
	length := p.MhLength
	if p.MhType == mh.IDENTITY {
		length = -1
	}

	if p.Version == 0 && (p.MhType != mh.SHA2_256 ||
		(p.MhLength != 32 && p.MhLength != -1)) {

		return Undef, ErrInvalidCid{fmt.Errorf("invalid v0 prefix")}
	}

	hash, err := mh.Sum(data, p.MhType, length)
	if err != nil {
		return Undef, ErrInvalidCid{err}
	}

	switch p.Version {
	case 0:
		return NewCidV0(hash), nil
	case 1:
		return NewCidV1(p.Codec, hash), nil
	default:
		return Undef, ErrInvalidCid{fmt.Errorf("invalid cid version")}
	}
}

// Bytes returns a byte representation of a Prefix. It looks like:
//
//	<version><codec><mh-type><mh-length>
func (p Prefix) Bytes() []byte {
	size := varint.UvarintSize(p.Version)
	size += varint.UvarintSize(p.Codec)
	size += varint.UvarintSize(p.MhType)
	size += varint.UvarintSize(uint64(p.MhLength))

	buf := make([]byte, size)
	n := varint.PutUvarint(buf, p.Version)
	n += varint.PutUvarint(buf[n:], p.Codec)
	n += varint.PutUvarint(buf[n:], p.MhType)
	n += varint.PutUvarint(buf[n:], uint64(p.MhLength))
	if n != size {
		panic("size mismatch")
	}
	return buf
}

// PrefixFromBytes parses a Prefix-byte representation onto a
// Prefix.
func PrefixFromBytes(buf []byte) (Prefix, error) {
	r := bytes.NewReader(buf)
	vers, err := varint.ReadUvarint(r)
	if err != nil {
		return Prefix{}, ErrInvalidCid{err}
	}

	codec, err := varint.ReadUvarint(r)
	if err != nil {
		return Prefix{}, ErrInvalidCid{err}
	}

	mhtype, err := varint.ReadUvarint(r)
	if err != nil {
		return Prefix{}, ErrInvalidCid{err}
	}

	mhlen, err := varint.ReadUvarint(r)
	if err != nil {
		return Prefix{}, ErrInvalidCid{err}
	}

	return Prefix{
		Version:  vers,
		Codec:    codec,
		MhType:   mhtype,
		MhLength: int(mhlen),
	}, nil
}

func CidFromBytes(data []byte) (int, Cid, error) {
	return CidFromBytesOf[string](data)
}

func CidFromBytesOf[S Storage](data []byte) (int, CidOf[S], error) {
	var zero CidOf[S]
	if len(data) > 2 && data[0] == mh.SHA2_256 && data[1] == 32 {
		if len(data) < 34 {
			return 0, zero, ErrInvalidCid{fmt.Errorf("not enough bytes for cid v0")}
		}

		h, err := mh.Cast(data[:34])
		if err != nil {
			return 0, zero, ErrInvalidCid{err}
		}

		return 34, CidOf[S]{S(h)}, nil
	}

	vers, n, err := varint.FromUvarint(data)
	if err != nil {
		return 0, zero, ErrInvalidCid{err}
	}

	if vers != 1 {
		return 0, zero, ErrInvalidCid{fmt.Errorf("expected 1 as the cid version number, got: %d", vers)}
	}

	_, cn, err := varint.FromUvarint(data[n:])
	if err != nil {
		return 0, zero, ErrInvalidCid{err}
	}

	mhnr, _, err := mh.MHFromBytes(data[n+cn:])
	if err != nil {
		return 0, zero, ErrInvalidCid{err}
	}

	l := n + cn + mhnr

	return l, CidOf[S]{S(data[:l])}, nil
}

func toBufByteReader(r io.Reader, dst []byte) *bufByteReader {
	// If the reader already implements ByteReader, use it directly.
	// Otherwise, use a fallback that does 1-byte Reads.
	if br, ok := r.(io.ByteReader); ok {
		return &bufByteReader{direct: br, dst: dst}
	}
	return &bufByteReader{fallback: r, dst: dst}
}

type bufByteReader struct {
	direct   io.ByteReader
	fallback io.Reader

	dst []byte
}

func (r *bufByteReader) ReadByte() (byte, error) {
	// The underlying reader has ReadByte; use it.
	if br := r.direct; br != nil {
		b, err := br.ReadByte()
		if err != nil {
			return 0, err
		}
		r.dst = append(r.dst, b)
		return b, nil
	}

	// Fall back to a one-byte Read.
	// TODO: consider reading straight into dst,
	// once we have benchmarks and if they prove that to be faster.
	var p [1]byte
	if _, err := io.ReadFull(r.fallback, p[:]); err != nil {
		return 0, err
	}
	r.dst = append(r.dst, p[0])
	return p[0], nil
}

// CidFromReader reads a precise number of bytes for a CID from a given reader.
// It returns the number of bytes read, the CID, and any error encountered.
// The number of bytes read is accurate even if a non-nil error is returned.
//
// It's recommended to supply a reader that buffers and implements io.ByteReader,
// as CidFromReader has to do many single-byte reads to decode varints.
// If the argument only implements io.Reader, single-byte Read calls are used instead.
//
// If the Reader is found to yield zero bytes, an io.EOF error is returned directly, in all
// other error cases, an ErrInvalidCid, wrapping the original error, is returned.
func CidFromReader(r io.Reader) (int, Cid, error) {
	return CidFromReaderOf[string](r)
}

// CidFromReaderOf is like [CidFromReader].
func CidFromReaderOf[S Storage](r io.Reader) (int, CidOf[S], error) {
	var zero CidOf[S]
	// 64 bytes is enough for any CIDv0,
	// and it's enough for most CIDv1s in practice.
	// If the digest is too long, we'll allocate more.
	br := toBufByteReader(r, make([]byte, 0, 64))

	// We read the first varint, to tell if this is a CIDv0 or a CIDv1.
	// The varint package wants a io.ByteReader, so we must wrap our io.Reader.
	vers, err := varint.ReadUvarint(br)
	if err != nil {
		if err == io.EOF {
			// First-byte read in ReadUvarint errors with io.EOF, so reader has no data.
			// Subsequent reads with an EOF will return io.ErrUnexpectedEOF and be wrapped here.
			return 0, zero, err
		}
		return len(br.dst), zero, ErrInvalidCid{err}
	}

	// If we have a CIDv0, read the rest of the bytes and cast the buffer.
	if vers == mh.SHA2_256 {
		if n, err := io.ReadFull(r, br.dst[1:34]); err != nil {
			return len(br.dst) + n, zero, ErrInvalidCid{err}
		}

		br.dst = br.dst[:34]
		h, err := mh.Cast(br.dst)
		if err != nil {
			return len(br.dst), zero, ErrInvalidCid{err}
		}

		return len(br.dst), CidOf[S]{S(h)}, nil
	}

	if vers != 1 {
		return len(br.dst), zero, ErrInvalidCid{fmt.Errorf("expected 1 as the cid version number, got: %d", vers)}
	}

	// CID block encoding multicodec.
	_, err = varint.ReadUvarint(br)
	if err != nil {
		return len(br.dst), zero, ErrInvalidCid{err}
	}

	// We could replace most of the code below with go-multihash's ReadMultihash.
	// Note that it would save code, but prevent reusing buffers.
	// Plus, we already have a ByteReader now.
	mhStart := len(br.dst)

	// Multihash hash function code.
	_, err = varint.ReadUvarint(br)
	if err != nil {
		return len(br.dst), zero, ErrInvalidCid{err}
	}

	// Multihash digest length.
	mhl, err := varint.ReadUvarint(br)
	if err != nil {
		return len(br.dst), zero, ErrInvalidCid{err}
	}

	// Refuse to make large allocations to prevent OOMs due to bugs.
	const maxDigestAlloc = 32 << 20 // 32MiB
	if mhl > maxDigestAlloc {
		return len(br.dst), zero, ErrInvalidCid{fmt.Errorf("refusing to allocate %d bytes for a digest", mhl)}
	}

	// Fine to convert mhl to int, given maxDigestAlloc.
	prefixLength := len(br.dst)
	cidLength := prefixLength + int(mhl)
	if cidLength > cap(br.dst) {
		// If the multihash digest doesn't fit in our initial 64 bytes,
		// efficiently extend the slice via append+make.
		br.dst = append(br.dst, make([]byte, cidLength-len(br.dst))...)
	} else {
		// The multihash digest fits inside our buffer,
		// so just extend its capacity.
		br.dst = br.dst[:cidLength]
	}

	if n, err := io.ReadFull(r, br.dst[prefixLength:cidLength]); err != nil {
		// We can't use len(br.dst) here,
		// as we've only read n bytes past prefixLength.
		return prefixLength + n, zero, ErrInvalidCid{err}
	}

	// This simply ensures the multihash is valid.
	// TODO: consider removing this bit, as it's probably redundant;
	// for now, it helps ensure consistency with CidFromBytes.
	_, _, err = mh.MHFromBytes(br.dst[mhStart:])
	if err != nil {
		return len(br.dst), zero, ErrInvalidCid{err}
	}

	return len(br.dst), CidOf[S]{S(br.dst)}, nil
}
