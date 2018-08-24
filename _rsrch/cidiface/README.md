What golang Kinds work best to implement CIDs?
==============================================

There are many possible ways to implement CIDs.  This package explores them.

- Option A: CIDs as a struct; multihash as bytes.
- Option B: CIDs as a string.
- Option C: CIDs as an interface with multiple implementors.
- Option D: CIDs as a struct; multihash also as a struct or string.

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
