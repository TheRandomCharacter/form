### Result is returned by value
This clearly states that ownership is transfered to caller. If one is bothered
by performance penalty, one should carefully benchmark performance difference
between copying result allocated on stack and allocation, inderection during
usage and garbage collection on heap. If ones findings require result returned
by pointer, a pointer to desired decoded type is a supported type parameter.

### Building values prefix tree instead of iteratively decoding values
This has the benefit of avoiding copy while decoding composite map value
elements. In addition one can provide complex custom type decoders which take in
account all the source values with same prefix. Worse case prefix tree requires
aditional memory equal to the sum of sizes of values paths. In contrast
iterative decoding of slices has to overallocate memory or reallocate and copy
underlying array on each additional value provided.

### Specific values modify general ones
Decoder (if registered) for a composite member is triggered first, then modified
with specific values.

## TODO
### Set maximum slice length
	,max200,
