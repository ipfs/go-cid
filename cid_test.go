package cid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	mbase "github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

// Copying the "silly test" idea from
// https://github.com/multiformats/go-multihash/blob/7aa9f26a231c6f34f4e9fad52bf580fd36627285/multihash_test.go#L13
// Makes it so changing the table accidentally has to happen twice.
var tCodecs = map[uint64]string{
	Raw:                   "raw",
	DagProtobuf:           "protobuf",
	DagCBOR:               "cbor",
	Libp2pKey:             "libp2p-key",
	GitRaw:                "git-raw",
	EthBlock:              "eth-block",
	EthBlockList:          "eth-block-list",
	EthTxTrie:             "eth-tx-trie",
	EthTx:                 "eth-tx",
	EthTxReceiptTrie:      "eth-tx-receipt-trie",
	EthTxReceipt:          "eth-tx-receipt",
	EthStateTrie:          "eth-state-trie",
	EthAccountSnapshot:    "eth-account-snapshot",
	EthStorageTrie:        "eth-storage-trie",
	BitcoinBlock:          "bitcoin-block",
	BitcoinTx:             "bitcoin-tx",
	ZcashBlock:            "zcash-block",
	ZcashTx:               "zcash-tx",
	DecredBlock:           "decred-block",
	DecredTx:              "decred-tx",
	DashBlock:             "dash-block",
	DashTx:                "dash-tx",
	FilCommitmentUnsealed: "fil-commitment-unsealed",
	FilCommitmentSealed:   "fil-commitment-sealed",
	DagJOSE:               "dag-jose",
}

func assertEqual(t *testing.T, a, b Cid) {
	if a.Type() != b.Type() {
		t.Fatal("mismatch on type")
	}

	if a.Version() != b.Version() {
		t.Fatal("mismatch on version")
	}

	if !bytes.Equal(a.Hash(), b.Hash()) {
		t.Fatal("multihash mismatch")
	}
}

func TestTable(t *testing.T) {
	if len(tCodecs) != len(Codecs)-1 {
		t.Errorf("Item count mismatch in the Table of Codec. Should be %d, got %d", len(tCodecs)+1, len(Codecs))
	}

	for k, v := range tCodecs {
		if Codecs[v] != k {
			t.Errorf("Table mismatch: 0x%x %s", k, v)
		}
	}
}

// The table returns cid.DagProtobuf for "v0"
// so we test it apart
func TestTableForV0(t *testing.T) {
	if Codecs["v0"] != DagProtobuf {
		t.Error("Table mismatch: Codecs[\"v0\"] should resolve to DagProtobuf (0x70)")
	}
}

func TestPrefixSum(t *testing.T) {
	// Test creating CIDs both manually and with Prefix.
	// Tests: https://github.com/ipfs/go-cid/issues/83
	for _, hashfun := range []uint64{
		mh.IDENTITY, mh.SHA3, mh.SHA2_256,
	} {
		h1, err := mh.Sum([]byte("TEST"), hashfun, -1)
		if err != nil {
			t.Fatal(err)
		}
		c1 := NewCidV1(Raw, h1)

		h2, err := mh.Sum([]byte("foobar"), hashfun, -1)
		if err != nil {
			t.Fatal(err)
		}
		c2 := NewCidV1(Raw, h2)

		c3, err := c1.Prefix().Sum([]byte("foobar"))
		if err != nil {
			t.Fatal(err)
		}
		if !c2.Equals(c3) {
			t.Fatal("expected CIDs to be equal")
		}
	}
}

func TestBasicMarshaling(t *testing.T) {
	h, err := mh.Sum([]byte("TEST"), mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := NewCidV1(7, h)

	data := cid.Bytes()

	out, err := Cast(data)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cid, out)

	s := cid.String()
	out2, err := Decode(s)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cid, out2)
}

func TestBasesMarshaling(t *testing.T) {
	h, err := mh.Sum([]byte("TEST"), mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := NewCidV1(7, h)

	data := cid.Bytes()

	out, err := Cast(data)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cid, out)

	testBases := []mbase.Encoding{
		mbase.Base16,
		mbase.Base32,
		mbase.Base32hex,
		mbase.Base32pad,
		mbase.Base32hexPad,
		mbase.Base58BTC,
		mbase.Base58Flickr,
		mbase.Base64pad,
		mbase.Base64urlPad,
		mbase.Base64url,
		mbase.Base64,
	}

	for _, b := range testBases {
		s, err := cid.StringOfBase(b)
		if err != nil {
			t.Fatal(err)
		}

		if s[0] != byte(b) {
			t.Fatal("Invalid multibase header")
		}

		out2, err := Decode(s)
		if err != nil {
			t.Fatal(err)
		}

		assertEqual(t, cid, out2)

		encoder, err := mbase.NewEncoder(b)
		if err != nil {
			t.Fatal(err)
		}
		s2 := cid.Encode(encoder)
		if s != s2 {
			t.Fatalf("%q != %q", s, s2)
		}

		ee, err := ExtractEncoding(s)
		if err != nil {
			t.Fatal(err)
		}
		if ee != b {
			t.Fatalf("could not properly determine base (got %v)", ee)
		}
	}

	ee, err := ExtractEncoding("QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n")
	if err != nil {
		t.Fatal(err)
	}
	if ee != mbase.Base58BTC {
		t.Fatalf("expected Base58BTC from Qm string (got %v)", ee)
	}

	ee, err = ExtractEncoding("1")
	if err == nil {
		t.Fatal("expected too-short error from ExtractEncoding")
	}
	if ee != -1 {
		t.Fatal("expected -1 from too-short ExtractEncoding")
	}
}

func TestBinaryMarshaling(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(DagCBOR, hash)
	var c2 Cid
	var c3 Cid

	data, err := c.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err = c2.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !c.Equals(c2) {
		t.Errorf("cids should be the same: %s %s", c, c2)
	}
	var buf bytes.Buffer
	wrote, err := c.WriteBytes(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if wrote != 36 {
		t.Fatalf("expected 36 bytes written (got %d)", wrote)
	}
	if err = c3.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if !c.Equals(c3) {
		t.Errorf("cids should be the same: %s %s", c, c3)
	}
}

func TestTextMarshaling(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(DagCBOR, hash)
	var c2 Cid

	data, err := c.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if err = c2.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if !c.Equals(c2) {
		t.Errorf("cids should be the same: %s %s", c, c2)
	}

	if c.KeyString() != string(c.Bytes()) {
		t.Errorf("got unexpected KeyString() result")
	}
}

func TestEmptyString(t *testing.T) {
	_, err := Decode("")
	if err == nil {
		t.Fatal("shouldnt be able to parse an empty cid")
	}
}

func TestV0Handling(t *testing.T) {
	old := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n"

	cid, err := Decode(old)
	if err != nil {
		t.Fatal(err)
	}

	if cid.Version() != 0 {
		t.Fatal("should have gotten version 0 cid")
	}

	if cid.Hash().B58String() != old {
		t.Fatalf("marshaling roundtrip failed: %s != %s", cid.Hash().B58String(), old)
	}

	if cid.String() != old {
		t.Fatal("marshaling roundtrip failed")
	}

	byteLen := cid.ByteLen()
	if byteLen != 34 {
		t.Fatalf("expected V0 ByteLen to be 34 (got %d)", byteLen)
	}

	new, err := cid.StringOfBase(mbase.Base58BTC)
	if err != nil {
		t.Fatal(err)
	}
	if new != old {
		t.Fatal("StringOfBase roundtrip failed")
	}

	encoder, err := mbase.NewEncoder(mbase.Base58BTC)
	if err != nil {
		t.Fatal(err)
	}
	if cid.Encode(encoder) != old {
		t.Fatal("Encode roundtrip failed")
	}

	_, err = cid.StringOfBase(mbase.Base32)
	if err != ErrInvalidEncoding {
		t.Fatalf("expected ErrInvalidEncoding for V0 StringOfBase(Base32) (got %v)", err)
	}
}

func TestV0ErrorCases(t *testing.T) {
	badb58 := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zIII"
	_, err := Decode(badb58)
	if err == nil {
		t.Fatal("should have failed to decode that ref")
	}
}

func TestNewPrefixV1(t *testing.T) {
	data := []byte("this is some test content")

	// Construct c1
	prefix := NewPrefixV1(DagCBOR, mh.SHA2_256)
	c1, err := prefix.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	if c1.Prefix() != prefix {
		t.Fatal("prefix not preserved")
	}

	// Construct c2
	hash, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCidV1(DagCBOR, hash)

	if !c1.Equals(c2) {
		t.Fatal("cids mismatch")
	}
	if c1.Prefix() != c2.Prefix() {
		t.Fatal("prefixes mismatch")
	}
}

func TestNewPrefixV0(t *testing.T) {
	data := []byte("this is some test content")

	// Construct c1
	prefix := NewPrefixV0(mh.SHA2_256)
	c1, err := prefix.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	if c1.Prefix() != prefix {
		t.Fatal("prefix not preserved")
	}

	// Construct c2
	hash, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCidV0(hash)

	if !c1.Equals(c2) {
		t.Fatal("cids mismatch")
	}
	if c1.Prefix() != c2.Prefix() {
		t.Fatal("prefixes mismatch")
	}
}

func TestInvalidV0Prefix(t *testing.T) {
	tests := []Prefix{
		{
			MhType:   mh.SHA2_256,
			MhLength: 31,
		},
		{
			MhType:   mh.SHA2_256,
			MhLength: 33,
		},
		{
			MhType:   mh.SHA2_256,
			MhLength: -2,
		},
		{
			MhType:   mh.SHA2_512,
			MhLength: 32,
		},
		{
			MhType:   mh.SHA2_512,
			MhLength: -1,
		},
	}

	for i, p := range tests {
		t.Log(i)
		_, err := p.Sum([]byte("testdata"))
		if err == nil {
			t.Fatalf("should error (index %d)", i)
		}
	}
}

func TestBadPrefix(t *testing.T) {
	p := Prefix{Version: 3, Codec: DagProtobuf, MhType: mh.SHA2_256, MhLength: 3}
	_, err := p.Sum([]byte{0x00, 0x01, 0x03})
	if err == nil {
		t.Fatalf("expected error on v3 prefix Sum")
	}
}

func TestPrefixRoundtrip(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(DagCBOR, hash)

	pref := c.Prefix()

	c2, err := pref.Sum(data)
	if err != nil {
		t.Fatal(err)
	}

	if !c.Equals(c2) {
		t.Fatal("output didnt match original")
	}

	pb := pref.Bytes()

	pref2, err := PrefixFromBytes(pb)
	if err != nil {
		t.Fatal(err)
	}

	if pref.Version != pref2.Version || pref.Codec != pref2.Codec ||
		pref.MhType != pref2.MhType || pref.MhLength != pref2.MhLength {
		t.Fatal("input prefix didnt match output")
	}
}

func TestBadPrefixFromBytes(t *testing.T) {
	_, err := PrefixFromBytes([]byte{0x80})
	if err == nil {
		t.Fatal("expected error for bad byte 0")
	}
	_, err = PrefixFromBytes([]byte{0x01, 0x80})
	if err == nil {
		t.Fatal("expected error for bad byte 1")
	}
	_, err = PrefixFromBytes([]byte{0x01, 0x01, 0x80})
	if err == nil {
		t.Fatal("expected error for bad byte 2")
	}
	_, err = PrefixFromBytes([]byte{0x01, 0x01, 0x01, 0x80})
	if err == nil {
		t.Fatal("expected error for bad byte 3")
	}
}

func Test16BytesVarint(t *testing.T) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	c := NewCidV1(1<<63, hash)
	_ = c.Bytes()
}

func TestFuzzCid(t *testing.T) {
	buf := make([]byte, 128)
	for i := 0; i < 200; i++ {
		s := rand.Intn(128)
		rand.Read(buf[:s])
		_, _ = Cast(buf[:s])
	}
}

func TestParse(t *testing.T) {
	cid, err := Parse(123)
	if err == nil {
		t.Fatalf("expected error from Parse()")
	}
	if !strings.Contains(err.Error(), "can't parse 123 as Cid") {
		t.Fatalf("expected int error, got %s", err.Error())
	}

	theHash := "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n"
	h, err := mh.FromB58String(theHash)
	if err != nil {
		t.Fatal(err)
	}

	assertions := [][]interface{}{
		{NewCidV0(h), theHash},
		{NewCidV0(h).Bytes(), theHash},
		{h, theHash},
		{theHash, theHash},
		{"/ipfs/" + theHash, theHash},
		{"https://ipfs.io/ipfs/" + theHash, theHash},
		{"http://localhost:8080/ipfs/" + theHash, theHash},
	}

	assert := func(arg interface{}, expected string) error {
		cid, err = Parse(arg)
		if err != nil {
			return err
		}
		if cid.Version() != 0 {
			return fmt.Errorf("expected version 0, got %d", cid.Version())
		}
		actual := cid.Hash().B58String()
		if actual != expected {
			return fmt.Errorf("expected hash %s, got %s", expected, actual)
		}
		actual = cid.String()
		if actual != expected {
			return fmt.Errorf("expected string %s, got %s", expected, actual)
		}
		return nil
	}

	for _, args := range assertions {
		if err := assert(args[0], args[1].(string)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHexDecode(t *testing.T) {
	hexcid := "f015512209d8453505bdc6f269678e16b3e56c2a2948a41f2c792617cc9611ed363c95b63"
	c, err := Decode(hexcid)
	if err != nil {
		t.Fatal(err)
	}

	if c.String() != "bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm" {
		t.Fatal("hash value failed to round trip decoding from hex")
	}
}

func ExampleDecode() {
	encoded := "bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm"
	c, err := Decode(encoded)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	fmt.Println(c)
	// Output: bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm
}

func TestFromJson(t *testing.T) {
	cval := "bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm"
	jsoncid := []byte(`{"/":"` + cval + `"}`)
	var c Cid
	if err := json.Unmarshal(jsoncid, &c); err != nil {
		t.Fatal(err)
	}

	if c.String() != cval {
		t.Fatal("json parsing failed")
	}
}

func TestJsonRoundTrip(t *testing.T) {
	expectedJSON := `{"/":"bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm"}`
	exp, err := Decode("bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm")
	if err != nil {
		t.Fatal(err)
	}

	// Verify it works for a *Cid.
	enc, err := json.Marshal(exp)
	if err != nil {
		t.Fatal(err)
	}
	var actual Cid
	if err = json.Unmarshal(enc, &actual); err != nil {
		t.Fatal(err)
	}
	if !exp.Equals(actual) {
		t.Fatal("cids not equal for *Cid")
	}

	if string(enc) != expectedJSON {
		t.Fatalf("did not get expected JSON form (got %q)", string(enc))
	}

	// Verify it works for a Cid.
	enc, err = json.Marshal(exp)
	if err != nil {
		t.Fatal(err)
	}
	var actual2 Cid
	if err = json.Unmarshal(enc, &actual2); err != nil {
		t.Fatal(err)
	}
	if !exp.Equals(actual2) {
		t.Fatal("cids not equal for Cid")
	}

	if err = actual2.UnmarshalJSON([]byte("1")); err == nil {
		t.Fatal("expected error for too-short JSON")
	}

	if err = actual2.UnmarshalJSON([]byte(`{"nope":"nope"}`)); err == nil {
		t.Fatal("expected error for bad CID JSON")
	}

	if err = actual2.UnmarshalJSON([]byte(`bad "" json!`)); err == nil {
		t.Fatal("expected error for bad JSON")
	}

	var actual3 Cid
	enc, err = actual3.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(enc) != "null" {
		t.Fatalf("expected 'null' string for undefined CID (got %q)", string(enc))
	}
}

func BenchmarkStringV1(b *testing.B) {
	data := []byte("this is some test content")
	hash, _ := mh.Sum(data, mh.SHA2_256, -1)
	cid := NewCidV1(Raw, hash)

	b.ReportAllocs()
	b.ResetTimer()

	count := 0
	for i := 0; i < b.N; i++ {
		count += len(cid.String())
	}
	if count != 49*b.N {
		b.FailNow()
	}
}

func TestReadCidsFromBuffer(t *testing.T) {
	cidstr := []string{
		"bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm",
		"k2cwueckqkibutvhkr4p2ln2pjcaxaakpd9db0e7j7ax1lxhhxy3ekpv",
		"Qmf5Qzp6nGBku7CEn2UQx4mgN8TW69YUok36DrGa6NN893",
		"zb2rhZi1JR4eNc2jBGaRYJKYM8JEB4ovenym8L1CmFsRAytkz",
		"bafkqarjpmzuwyzltorxxezjpkvcfgqkfjfbfcvslivje2vchkzdu6rckjjcfgtkolaze6mssjqzeyn2ekrcfatkjku2vowseky3fswkfkm2deqkrju3e2",
	}

	var cids []Cid
	var buf []byte
	for _, cs := range cidstr {
		c, err := Decode(cs)
		if err != nil {
			t.Fatal(err)
		}
		cids = append(cids, c)
		buf = append(buf, c.Bytes()...)
	}

	var cur int
	for _, expc := range cids {
		n, c, err := CidFromBytes(buf[cur:])
		if err != nil {
			t.Fatal(err)
		}
		if c != expc {
			t.Fatal("cids mismatched")
		}
		cur += n
	}
	if cur != len(buf) {
		t.Fatal("had trailing bytes")
	}

	// The same, but now with CidFromReader.
	// In multiple forms, to catch more io interface bugs.
	for _, r := range []io.Reader{
		// implements io.ByteReader
		bytes.NewReader(buf),

		// tiny reads, no io.ByteReader
		iotest.OneByteReader(bytes.NewReader(buf)),
	} {
		cur = 0
		for _, expc := range cids {
			n, c, err := CidFromReader(r)
			if err != nil {
				t.Fatal(err)
			}
			if c != expc {
				t.Fatal("cids mismatched")
			}
			cur += n
		}
		if cur != len(buf) {
			t.Fatal("had trailing bytes")
		}
	}
}

func TestBadCidInput(t *testing.T) {
	for _, name := range []string{
		"FromBytes",
		"FromReader",
	} {
		t.Run(name, func(t *testing.T) {
			usingReader := name == "FromReader"

			fromBytes := CidFromBytes
			if usingReader {
				fromBytes = func(data []byte) (int, Cid, error) {
					return CidFromReader(bytes.NewReader(data))
				}
			}

			l, c, err := fromBytes([]byte{mh.SHA2_256, 32, 0x00})
			if err == nil {
				t.Fatal("expected not-enough-bytes for V0 CID")
			}
			if !usingReader && l != 0 {
				t.Fatal("expected length==0 from bad CID")
			} else if usingReader && l == 0 {
				t.Fatal("expected length!=0 from bad CID")
			}
			if c != Undef {
				t.Fatal("expected Undef CID from bad CID")
			}

			c, err = Decode("bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm")
			if err != nil {
				t.Fatal(err)
			}
			byts := make([]byte, c.ByteLen())
			copy(byts, c.Bytes())
			byts[1] = 0x80 // bad codec varint
			byts[2] = 0x00
			l, c, err = fromBytes(byts)
			if err == nil {
				t.Fatal("expected not-enough-bytes for V1 CID")
			}
			if !usingReader && l != 0 {
				t.Fatal("expected length==0 from bad CID")
			} else if usingReader && l == 0 {
				t.Fatal("expected length!=0 from bad CID")
			}
			if c != Undef {
				t.Fatal("expected Undef CID from bad CID")
			}

			copy(byts, c.Bytes())
			byts[2] = 0x80 // bad multihash varint
			byts[3] = 0x00
			l, c, err = fromBytes(byts)
			if err == nil {
				t.Fatal("expected not-enough-bytes for V1 CID")
			}
			if !usingReader && l != 0 {
				t.Fatal("expected length==0 from bad CID")
			} else if usingReader && l == 0 {
				t.Fatal("expected length!=0 from bad CID")
			}
			if c != Undef {
				t.Fatal("expected Undef CID from bad CidFromBytes")
			}
		})
	}
}

func TestBadParse(t *testing.T) {
	hash, err := mh.Sum([]byte("foobar"), mh.SHA3_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(hash)
	if err == nil {
		t.Fatal("expected to fail to parse an invalid CIDv1 CID")
	}
}

func TestLoggable(t *testing.T) {
	c, err := Decode("bafkreie5qrjvaw64n4tjm6hbnm7fnqvcssfed4whsjqxzslbd3jwhsk3mm")
	if err != nil {
		t.Fatal(err)
	}
	actual := c.Loggable()
	expected := make(map[string]interface{})
	expected["cid"] = c
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("did not get expected loggable form (got %v)", actual)
	}
}
