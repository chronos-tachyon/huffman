package huffman

import (
	"math"
)

// Symbol represents a symbol in an arbitrary alphabet.  Negative symbols are
// not valid.
type Symbol int32

// MaxSymbol is the maximum valid symbol.
const MaxSymbol = Symbol(math.MaxInt32)

// InvalidSymbol is returned by some functions to clearly indicate that no
// symbol is being returned.
const InvalidSymbol = Symbol(-1)
