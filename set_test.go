package cid

import (
	"crypto/rand"
	"errors"
	"testing"

	mh "github.com/multiformats/go-multihash"
)

func makeRandomCid(t *testing.T) *Cid {
	p := make([]byte, 256)
	_, err := rand.Read(p)
	if err != nil {
		t.Fatal(err)
	}

	h, err := mh.Sum(p, mh.SHA3, 4)
	if err != nil {
		t.Fatal(err)
	}

	cid := &Cid{
		codec:   7,
		version: 1,
		hash:    h,
	}

	return cid
}

func TestSet(t *testing.T) {
	cid := makeRandomCid(t)
	cid2 := makeRandomCid(t)
	s := NewSet()

	s.Add(cid)

	if !s.Has(cid) {
		t.Error("should have the CID")
	}

	if s.Len() != 1 {
		t.Error("should report 1 element")
	}

	keys := s.Keys()

	if len(keys) != 1 || !keys[0].Equals(cid) {
		t.Error("key should correspond to Cid")
	}

	if s.Visit(cid) {
		t.Error("visit should return false")
	}

	foreach := []*Cid{}
	foreachF := func(c *Cid) error {
		foreach = append(foreach, c)
		return nil
	}

	if err := s.ForEach(foreachF); err != nil {
		t.Error(err)
	}

	if len(foreach) != 1 {
		t.Error("ForEach should have visited 1 element")
	}

	foreachErr := func(c *Cid) error {
		return errors.New("test")
	}

	if err := s.ForEach(foreachErr); err == nil {
		t.Error("Should have returned an error")
	}

	if !s.Visit(cid2) {
		t.Error("should have visited a new Cid")
	}

	if s.Len() != 2 {
		t.Error("len should be 2 now")
	}

	s.Remove(cid2)

	if s.Len() != 1 {
		t.Error("len should be 1 now")
	}
}

func TestSetPair(t *testing.T) {
	cid := makeRandomCid(t)
	cid2 := makeRandomCid(t)
	s := NewSet()

	visitF := func(curVal, newVal interface{}) bool {
		return curVal.(int) < newVal.(int)
	}

	s.AddPair(cid, 10)
	visited := s.VisitPair(cid, 5, visitF)
	if visited {
		t.Error("should not have visited the Cid since 5 <= 10")
	}

	visited = s.VisitPair(cid2, 3, visitF)
	if !visited {
		t.Error("should have visited a new value")
	}

	visited = s.VisitPair(cid, 11, visitF)
	if !visited {
		t.Error("should have visited because 10<11")
	}

	if s.Len() != 2 {
		t.Error("set should have 2 entries")
	}

	sum := 0
	forEachF := func(c *Cid, v interface{}) error {
		sum += v.(int)
		return nil
	}
	s.ForEachPair(forEachF)
	if sum != 14 {
		t.Error("items in set should total 3+11 = 14")
	}
}
