package cid

// Set is a implementation of a set of Cids, that is, a structure
// to which holds a single copy of every Cids that is added to it.
// It also works with pairs formed by a Cid and a user provided arbitrary
// value.
type Set struct {
	set map[string]interface{}
}

// NewSet initializes and returns a new Set.
func NewSet() *Set {
	return &Set{set: make(map[string]interface{})}
}

// Add puts a Cid in the Set.
func (s *Set) Add(c *Cid) {
	s.AddPair(c, struct{}{})
}

// AddPair puts a Cid and a custom value in the Set.
func (s *Set) AddPair(c *Cid, val interface{}) {
	s.set[string(c.Bytes())] = val
}

// Has returns if the Set contains a given Cid.
func (s *Set) Has(c *Cid) bool {
	_, ok := s.set[string(c.Bytes())]
	return ok
}

// Remove deletes a Cid from the Set.
func (s *Set) Remove(c *Cid) {
	delete(s.set, string(c.Bytes()))
}

// Len returns how many elements the Set has.
func (s *Set) Len() int {
	return len(s.set)
}

// Keys returns the Cids in the set.
func (s *Set) Keys() []*Cid {
	out := make([]*Cid, 0, len(s.set))
	for k := range s.set {
		c, _ := Cast([]byte(k))
		out = append(out, c)
	}
	return out
}

// Visit adds a Cid to the set only if it is
// not in it already.
func (s *Set) Visit(c *Cid) bool {
	visitF := func(curVal, newVal interface{}) bool {
		return false // never overwrite
	}

	return s.VisitPair(c, struct{}{}, visitF)
}

// VisitPair adds a pair to the set if the Cid is not already
// in the set OR the provided visitF function returns true.
// In other words, it replaces an existing pair if visitF returns true.
func (s *Set) VisitPair(c *Cid, newVal interface{}, visitF func(curVal, newVal interface{}) bool) bool {
	curVal, ok := s.set[string(c.Bytes())]
	if !ok || visitF(curVal, newVal) {
		s.AddPair(c, newVal)
		return true
	}
	return false

}

// ForEach allows to run a custom function on each
// Cid in the set.
func (s *Set) ForEach(f func(c *Cid) error) error {
	pairF := func(c *Cid, v interface{}) error {
		return f(c)
	}
	return s.ForEachPair(pairF)
}

// ForEachPair allows to run a custom function on each
// Pair in the set.
func (s *Set) ForEachPair(f func(c *Cid, v interface{}) error) error {
	for cs, v := range s.set {
		c, _ := Cast([]byte(cs))
		err := f(c, v)
		if err != nil {
			return err
		}
	}
	return nil
}
