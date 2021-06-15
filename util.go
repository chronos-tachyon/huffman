package huffman

import (
	mathbits "math/bits"
)

func log2uint32(x uint32) uint32 {
	if x == 0 {
		x = 1
	}
	return uint32(32 - mathbits.LeadingZeros32(x))
}
