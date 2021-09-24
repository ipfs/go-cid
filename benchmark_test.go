package cid_test

import (
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"math/rand"
	"testing"
)

// randSourceSeed represents the static RNG seed, fixed for reproducibility.
const randSourceSeed = 1413

// BenchmarkCidV1IdentityCheckUsingPrefix benchmarks checking whether a CIDv1 has multihash.IDENTITY
// code via. Cid.Prefix().
func BenchmarkCidV1IdentityCheckUsingPrefix(b *testing.B) {
	rng := rand.New(rand.NewSource(randSourceSeed))
	cv1 := generateCidV1(b, rng, multihash.IDENTITY)
	benchmarkCid(b, cv1, func(pb *testing.PB) {
		for pb.Next() {
			if cv1.Prefix().MhType != multihash.IDENTITY {
				b.Fatal("expected IDENTITY CID")
			}
		}
	})
}

// BenchmarkCidV1IdentityCheckUsingMultihashDecode benchmarks checking whether a CIDv1 has multihash.IDENTITY
// code via. decoding the Cid.Hash().
func BenchmarkCidV1IdentityCheckUsingMultihashDecode(b *testing.B) {
	rng := rand.New(rand.NewSource(randSourceSeed))
	cv1 := generateCidV1(b, rng, multihash.IDENTITY)
	benchmarkCid(b, cv1, func(pb *testing.PB) {
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
}

// BenchmarkCidV1IdentityCheckUsingIsIdentity benchmarks checking whether a CIDv1 has multihash.IDENTITY
// code via. decoding the Cid.IsIdentity().
func BenchmarkCidV1IdentityCheckUsingIsIdentity(b *testing.B) {
	rng := rand.New(rand.NewSource(randSourceSeed))
	cv1 := generateCidV1(b, rng, multihash.IDENTITY)
	benchmarkCid(b, cv1, func(pb *testing.PB) {
		for pb.Next() {
			if okId, err := cv1.IsIdentity(); err != nil || !okId {
				b.Fatal("expected IDENTITY CID")
			}
		}
	})
}

func benchmarkCid(b *testing.B, target cid.Cid, pb func(pb *testing.PB)) {
	b.SetBytes(int64(target.ByteLen()))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(pb)
}

func generateMultihash(b *testing.B, rng *rand.Rand, mhCode uint64) multihash.Multihash {
	// Generate random data to hash.
	data := make([]byte, rng.Intn(100)+1024)
	if _, err := rng.Read(data); err != nil {
		b.Fatal(err)
	}
	// Generate multihash from data.
	mh, err := multihash.Sum(data, mhCode, -1)
	if err != nil {
		b.Fatal(err)
	}
	return mh
}

func generateCidV1(b *testing.B, rng *rand.Rand, mhCode uint64) cid.Cid {
	mh := generateMultihash(b, rng, mhCode)
	return cid.NewCidV1(cid.Raw, mh)
}
