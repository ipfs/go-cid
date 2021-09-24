package cid_test

import (
	"math/rand"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// BenchmarkIdentityCheck benchmarks two ways of checking whether a CIDv1 has multihash.IDENTITY
// code:
//  1. Cid.Prefix(), and
//  2. decoding the Cid.Hash().
//
// This benchmark illustrates that using Cid.Prefix is more efficient than multihash.Decode.
// Users wishing to perform such a check should use Cid.Prefix.
//
// Consider that `Cid.Prefix` is already efficient enough and introducing a dedicated API for
// performing this check will likely result in small gains.
func BenchmarkIdentityCheck(b *testing.B) {
	rng := rand.New(rand.NewSource(1413))

	data := make([]byte, rng.Intn(100)+1024)
	if _, err := rng.Read(data); err != nil {
		b.Fatal(err)
	}
	mh, err := multihash.Sum(data, multihash.IDENTITY, -1)
	if err != nil {
		b.Fatal(err)
	}
	cv1 := cid.NewCidV1(cid.Raw, mh)

	b.SetBytes(int64(cv1.ByteLen()))
	b.ReportAllocs()
	b.ResetTimer()

	b.Run("Prefix", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if cv1.Prefix().MhType != multihash.IDENTITY {
					b.Fatal("expected IDENTITY CID")
				}
			}
		})
	})

	b.Run("MultihashDecode", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				dmh, err := multihash.Decode(cv1.Hash())
				if err != nil {
					b.Fatal(err)
				}
				if dmh.Code != multihash.IDENTITY {
					b.Fatal("expected IDENTITY CID")
				}
			}
		})
	})
}
