package huffman

import (
	"fmt"
	mathbits "math/bits"
	"strconv"
)

// Code represents a sequence of bits.
type Code struct {
	// Size holds the number of valid bits.
	Size byte

	// Bits holds the actual values of the bits.  The least significant bit
	// of Bits is the first bit.
	Bits uint32
}

// MakeCode is a convenience function that constructs a Code.
func MakeCode(size byte, bits uint32) Code {
	return Code{Size: size, Bits: bits}
}

// MakeReversedCode constructs a Code from a sequence of bits that's in the
// wrong order, i.e. the least significant bit is the *last* bit in the
// sequence, instead of the first.
func MakeReversedCode(size byte, bits uint32) Code {
	return MakeCode(size, reverseBits(size, bits))
}

// Reversed returns the corresponding Code with the bits in reverse order.
func (hc Code) Reversed() Code {
	return MakeReversedCode(hc.Size, hc.Bits)
}

// String returns the string representation of this Code.
func (hc Code) String() string {
	if hc.Size == 0 {
		return "\"\""
	}
	format := "%0" + strconv.FormatUint(uint64(hc.Size), 10) + "b"
	return strconv.Quote(fmt.Sprintf(format, hc.Bits))
}

var _ fmt.Stringer = Code{}

func reverseBits(size byte, bits uint32) uint32 {
	return mathbits.Reverse32(bits) >> (32 - size)
}
