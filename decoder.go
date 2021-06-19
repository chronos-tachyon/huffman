package huffman

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Decoder implements a decoder for canonical Huffman codes.
type Decoder struct {
	table   map[Code]decoderData
	sizes   []byte
	minSize byte
	maxSize byte
}

// NewDecoder is a convenience function that allocates a new Decoder and calls
// Init on it.  If Init returns an error, NewDecoder panics.
func NewDecoder(sizes []byte) *Decoder {
	d := new(Decoder)
	if err := d.Init(sizes); err != nil {
		panic(err)
	}
	return d
}

// Init initializes this Decoder.  The argument consists of zero or more bit
// lengths, one for each symbol in the code, which is used to construct the
// canonical Huffman code per the algorithm in RFC 1951 Section 3.2.2.  Symbols
// with an assigned bit length of 0 are omitted from the code entirely.
//
// Not all inputs are valid for constructing a canonical Huffman code.  In
// particular, this method will reject "degenerate" codes which use overly long
// big lengths for some inputs.  Degenerate codes consisting of 0 valid symbols
// or 1 valid symbol are permitted, however, as there is no way to construct a
// non-degenerate Huffman code for such cases.
//
func (d *Decoder) Init(sizes []byte) error {
	numSymbols := Symbol(len(sizes))

	var countArray [maxBitsPerCode]uint32
	var numSymbolsWithNonZeroSizes uint32
	var minSize, maxSize byte
	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		size := sizes[symbol]
		if size == 0 {
			continue
		}

		// forbid codes with sizes greater than maxBitsPerCode
		if size > maxBitsPerCode {
			return fmt.Errorf("invalid bit length while constructing Huffman tree: got %d, max %d", size, maxBitsPerCode)
		}

		if numSymbolsWithNonZeroSizes == 0 {
			minSize = size
			maxSize = size
		} else if minSize > size {
			minSize = size
		} else if maxSize < size {
			maxSize = size
		}

		countArray[size]++
		numSymbolsWithNonZeroSizes++
	}

	// permit degenerate code with 0 symbols
	if numSymbolsWithNonZeroSizes == 0 {
		*d = Decoder{}
		return nil
	}

	var nextCodeArray [maxBitsPerCode]uint32
	var code uint32
	for bits := minSize; bits <= maxSize; bits++ {
		code = (code + countArray[bits-1]) << 1
		nextCodeArray[bits] = code
	}
	code += countArray[maxSize]

	// permit degenerate code with 1 symbol
	// forbid all other degenerate codes
	if code == 1 && maxSize == 1 {
		// pass
	} else if code != (1 << maxSize) {
		return fmt.Errorf("degenerate Huffman tree: expected %d, got %d", (1 << maxSize), code)
	}

	// len(table) is approximately n×log2(n) when filled.
	numTableSlots := numSymbolsWithNonZeroSizes * log2uint32(numSymbolsWithNonZeroSizes)

	*d = Decoder{
		table:   make(map[Code]decoderData, numTableSlots),
		sizes:   make([]byte, numSymbols),
		minSize: minSize,
		maxSize: maxSize,
	}

	copy(d.sizes, sizes)

	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		size := sizes[symbol]
		if size == 0 {
			continue
		}

		code := nextCodeArray[size]
		nextCodeArray[size]++

		hc := MakeReversedCode(size, code)
		fillTable(d.table, symbol, hc)
	}

	return nil
}

// InitFromEncoder initializes this Decoder to be the mirror of the given
// Encoder.
func (d *Decoder) InitFromEncoder(e Encoder) error {
	return d.Init(e.SizeBySymbol())
}

// Decode attempts to decode a Huffman code into a Symbol.
//
// If the Decode is completely successful, symbol >= 0 and minSize == maxSize.
//
// If the Decode fails due to insufficient bits, symbol == InvalidSymbol and at
// least (minSize - hc.Size) additional bits are required to decode this
// symbol.  No more than (maxSize - hc.Size) additional bits will be required.
//
// If the Decode fails due to unreasonable input, symbol == InvalidSymbol and
// minSize == maxSize == 0.
//
func (d Decoder) Decode(hc Code) (symbol Symbol, minSize byte, maxSize byte) {
	dd, found := d.table[hc]
	if !found {
		return InvalidSymbol, 0, 0
	}
	return dd.symbol, dd.minSize, dd.maxSize
}

// MinSize is the bit length of the shortest legal code.
func (d Decoder) MinSize() byte {
	return d.minSize
}

// MaxSize is the bit length of the longest legal code.
func (d Decoder) MaxSize() byte {
	return d.maxSize
}

// NumSymbols returns the total number of symbols in the code's alphabet.
func (d Decoder) NumSymbols() uint {
	return uint(len(d.sizes))
}

// MaxSymbol is the last Symbol in the code's alphabet.
//
// (The first Symbol in the code's alphabet is always 0.)
//
func (d Decoder) MaxSymbol() Symbol {
	return Symbol(len(d.sizes)) - 1
}

// SizeBySymbol returns a copy of the original bit length array used to
// initialize this Decoder.
func (d Decoder) SizeBySymbol() []byte {
	return d.sizes
}

// Encoder returns a new Encoder which mirrors this Decoder.
func (d Decoder) Encoder() *Encoder {
	e := new(Encoder)
	if err := e.InitFromDecoder(d); err != nil {
		panic(err)
	}
	return e
}

// Dump writes DebugString() to the given writer.
func (d Decoder) Dump(w io.Writer) (int64, error) {
	r := strings.NewReader(d.DebugString())
	return r.WriteTo(w)
}

// DebugString returns a programmer-readable debugging string of the Decoder's
// current state.
func (d Decoder) DebugString() string {
	var buf strings.Builder
	buf.WriteString("Decoder{\n")
	fmt.Fprintf(&buf, "\tMinSize() = %d\n", d.minSize)
	fmt.Fprintf(&buf, "\tMaxSize() = %d\n", d.maxSize)
	keys := make(byCode, 0, len(d.table))
	for hc := range d.table {
		keys = append(keys, hc)
	}
	keys.Sort()
	for _, hc := range keys {
		dd := d.table[hc]
		fmt.Fprintf(&buf, "\tDecode(%s) = {%d, %d, %d}\n", hc, dd.symbol, dd.minSize, dd.maxSize)
	}
	buf.WriteString("}\n")
	return buf.String()
}

// GoString returns a Go expression that would reconstruct this Decoder.
func (d Decoder) GoString() string {
	var buf strings.Builder
	buf.WriteString("NewDecoder([]byte{")
	for index, size := range d.SizeBySymbol() {
		if index != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.FormatUint(uint64(size), 10))
	}
	buf.WriteString("})")
	return buf.String()
}

// String returns a brief string representation.
func (d Decoder) String() string {
	return fmt.Sprintf(
		"(Huffman decoder with %d symbols, with coded lengths of %d .. %d bits)",
		len(d.sizes),
		d.minSize,
		d.maxSize,
	)
}

// MarshalJSON renders this Decoder as JSON data.
func (d Decoder) MarshalJSON() ([]byte, error) {
	length := uint(len(d.sizes))
	arr := make([]uint, length)
	for i := uint(0); i < length; i++ {
		arr[i] = uint(d.sizes[i])
	}
	return json.Marshal(arr)
}

// UnmarshalJSON initializes this Decoder from JSON data.
func (d *Decoder) UnmarshalJSON(raw []byte) error {
	var arr []uint
	if err := json.Unmarshal(raw, &arr); err != nil {
		return err
	}

	length := uint(len(arr))
	sizes := make([]byte, length)
	for i := uint(0); i < length; i++ {
		size := arr[i]
		if size > maxBitsPerCode {
			return fmt.Errorf("invalid bit length while constructing Huffman tree: got %d, max %d", size, maxBitsPerCode)
		}
		sizes[i] = byte(size)
	}

	return d.Init(sizes)
}

type decoderData struct {
	symbol  Symbol
	minSize byte
	maxSize byte
}

func fillTable(table map[Code]decoderData, symbol Symbol, hc Code) {
	dd := decoderData{symbol, hc.Size, hc.Size}
	table[hc] = dd

	for hc.Size != 0 {
		// For each hc "axxx...", compute "Axxx..." where A = NOT a.

		bit := uint32(1) << (hc.Size - 1)
		hc.Bits ^= bit

		// Merge the dd's from "axxx..." (dd) and "Axxx..." (ddSibling)
		// into ddNew (the new parent for dd and ddSibling).

		ddNew := decoderData{InvalidSymbol, dd.minSize, dd.maxSize}
		if ddSibling, found := table[hc]; found {
			if ddNew.minSize > ddSibling.minSize {
				ddNew.minSize = ddSibling.minSize
			}
			if ddNew.maxSize < ddSibling.maxSize {
				ddNew.maxSize = ddSibling.maxSize
			}
		}

		// Mutate hc from "Axxx..." to "xxx...".

		hc.Size--
		hc.Bits &^= bit

		// If table[hc] already equals ddNew, we can stop recursing.

		if ddOld, found := table[hc]; found && ddOld == ddNew {
			break
		}

		// Update table[hc] with ddNew and continue recursing.

		table[hc] = ddNew
		dd = ddNew
	}
}

// type byCode {{{

type byCode []Code

func (list byCode) Sort() {
	sort.Sort(list)
}

func (list byCode) Len() int {
	return len(list)
}

func (list byCode) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list byCode) Less(i, j int) bool {
	a, b := list[i], list[j]
	as, ab := a.Size, a.Bits
	bs, bb := b.Size, b.Bits
	if as != bs {
		return as < bs
	}
	return ab < bb
}

var _ sort.Interface = byCode(nil)

// }}}
