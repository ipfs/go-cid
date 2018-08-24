What golang Kinds work best to implement CIDs?
==============================================

There are many possible ways to implement CIDs.  This package explores them.

### criteria

There's a couple different criteria to consider:

- We want the best performance when operating on the type (getters, mostly);
- We want to minimize the number of memory allocations we need;
- We want types which can be used as map keys, because this is common.

The priority of these criteria is open to argument, but it's probably
mapkeys > minalloc > anythingelse.
(Mapkeys and minalloc are also quite entangled, since if we don't pick a
representation that can work natively as a map key, we'll end up needing
a `KeyRepr()` method which gives us something that does work as a map key,
an that will almost certainly involve a malloc itself.)

### options

There are quite a few different ways to go:

- Option A: CIDs as a struct; multihash as bytes.
- Option B: CIDs as a string.
- Option C: CIDs as an interface with multiple implementors.
- Option D: CIDs as a struct; multihash also as a struct or string.
- Option E: CIDs as a struct; content as strings plus offsets.

The current approach on the master branch is Option A.

Option D is distinctive from Option A because multihash as bytes transitively
causes the CID struct to be non-comparible and thus not suitable for map keys
as per https://golang.org/ref/spec#KeyType .  (It's also a bit more work to
pursue Option D because it's just a bigger splash radius of change; but also,
something we might also want to do soon, because we *do* also have these same
map-key-usability concerns with multihash alone.)

Option E is distinctive from Option D because Option E would always maintain
the binary format of the cid internally, and so could yield it again without
malloc, while still potentially having faster access to components than
Option B since it wouldn't need to re-parse varints to access later fields.

Option C is the avoid-choices choice, but note that interfaces are not free;
since "minimize mallocs" is one of our major goals, we cannot use interfaces
whimsically.


Discoveries
-----------

### using interfaces as map keys forgoes a lot of safety checks

Using interfaces as map keys pushes a bunch of type checking to runtime.
E.g., it's totally valid at compile time to push a type which is non-comparable
into a map key; it will panic at *runtime* instead of failing at compile-time.

There's also no way to define equality checks between implementors of the
interface: golang will always use its innate concept of comparison for the
concrete types.  This means its effectively *never safe* to use two different
concrete implementations of an interface in the same map; you may add elements
which are semantically "equal" in your mind, and end up very confused later
when both impls of the same "equal" object have been stored.
