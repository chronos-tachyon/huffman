package huffman

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/chronos-tachyon/assert"
)

// Encoder implements an encoder for canonical Huffman codes.
type Encoder struct {
	codes   []Code
	minSize byte
	maxSize byte
}

// NewEncoder is a convenience function that allocates a new Encoder and calls
// Init on it.
func NewEncoder(numSymbols int, frequencies []uint32) *Encoder {
	e := new(Encoder)
	e.Init(numSymbols, frequencies)
	return e
}

// NewEncoderFromSizes is a convenience function that allocates a new Encoder
// and calls InitFromSizes on it.  If InitFromSizes returns an error,
// NewEncoderFromSizes panics.
func NewEncoderFromSizes(sizes []byte) *Encoder {
	e := new(Encoder)
	if err := e.InitFromSizes(sizes); err != nil {
		panic(err)
	}
	return e
}

// Init initializes this Encoder.  The first argument tells Init how many
// Symbols are in this code's alphabet, and the second argument lists the
// frequency (i.e. number of occurrences) for each Symbol in the code, one for
// each Symbol except that any Symbol not represented in the list is assumed to
// have a frequency of 0.
//
func (e *Encoder) Init(numSymbols int, frequencies []uint32) {
	assert.Assertf(numSymbols >= 1, "numSymbols %d < 1", numSymbols)
	assert.Assertf(numSymbols <= int(MaxSymbol), "numSymbols %d > MaxSymbol %d", numSymbols, int(MaxSymbol))
	assert.Assertf(numSymbols >= len(frequencies), "numSymbols %d < len(frequencies) %d", numSymbols, len(frequencies))

	codes := make([]Code, numSymbols)
	nodes := make([]symbolAndFreq, 0, numSymbols)
	for symbol := Symbol(0); symbol < Symbol(len(frequencies)); symbol++ {
		if freq := frequencies[symbol]; freq != 0 {
			nodes = append(nodes, symbolAndFreq{symbol, freq})
		}
	}

	var minSize, maxSize byte
	nodeLen := uint32(len(nodes))
	if nodeLen <= 2 {
		minSize, maxSize = 1, 1
		for index := uint32(0); index < nodeLen; index++ {
			node := nodes[index]
			codes[node.symbol] = MakeCode(1, index)
		}
	} else {
		firstPass(codes, nodes, &minSize, &maxSize)
		_ = secondPass(codes)
	}

	*e = Encoder{
		codes:   codes,
		minSize: minSize,
		maxSize: maxSize,
	}
}

// InitFromSizes initializes this Encoder from a list of bit lengths, one for
// each symbol in the code.  See Decoder.Init for more details.
func (e *Encoder) InitFromSizes(sizes []byte) error {
	numSymbols := Symbol(len(sizes))
	codes := make([]Code, numSymbols)

	var minSize, maxSize byte
	var hasMinMax bool

	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		size := sizes[symbol]
		if size == 0 {
			continue
		}

		if !hasMinMax {
			hasMinMax = true
			minSize = size
			maxSize = size
		} else if minSize > size {
			minSize = size
		} else if maxSize < size {
			maxSize = size
		}

		codes[symbol].Size = sizes[symbol]
	}

	if err := secondPass(codes); err != nil {
		return err
	}

	*e = Encoder{
		codes:   codes,
		minSize: minSize,
		maxSize: maxSize,
	}
	return nil
}

// InitFromDecoder initializes this Encoder to be the mirror of the given
// Decoder.
func (e *Encoder) InitFromDecoder(d Decoder) error {
	return e.InitFromSizes(d.SizeBySymbol())
}

// Encode encodes a Symbol into a Huffman-coded bit string.
func (e Encoder) Encode(symbol Symbol) Code {
	return e.codes[symbol]
}

// MinSize is the bit length of the shortest legal code.
func (e Encoder) MinSize() byte {
	return e.minSize
}

// MaxSize is the bit length of the longest legal code.
func (e Encoder) MaxSize() byte {
	return e.maxSize
}

// NumSymbols returns the total number of symbols in the code's alphabet.
func (e Encoder) NumSymbols() uint {
	return uint(len(e.codes))
}

// MaxSymbol is the last Symbol in the code's alphabet.
//
// (The first Symbol in the code's alphabet is always 0.)
//
func (e Encoder) MaxSymbol() Symbol {
	return Symbol(len(e.codes)) - 1
}

// SizeBySymbol returns an array containing the bit length for each Symbol in
// the alphabet.  This array can be transmitted to another party and used by
// Decoder to reconstruct this Huffman code on the receiving end.
//
func (e Encoder) SizeBySymbol() []byte {
	numSymbols := Symbol(len(e.codes))
	out := make([]byte, numSymbols)
	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		hc := e.codes[symbol]
		out[symbol] = hc.Size
	}
	return out
}

// Decoder returns a new Decoder which mirrors this Encoder.
func (e Encoder) Decoder() *Decoder {
	d := new(Decoder)
	if err := d.InitFromEncoder(e); err != nil {
		panic(err)
	}
	return d
}

// Dump writes DebugString() to the given writer.
func (e Encoder) Dump(w io.Writer) (int64, error) {
	r := strings.NewReader(e.DebugString())
	return r.WriteTo(w)
}

// DebugString returns a programmer-readable debugging string of the Encoder's
// current state.
func (e Encoder) DebugString() string {
	var buf strings.Builder
	buf.WriteString("Encoder{\n")
	fmt.Fprintf(&buf, "\tMinSize() = %d\n", e.minSize)
	fmt.Fprintf(&buf, "\tMaxSize() = %d\n", e.maxSize)
	numSymbols := Symbol(len(e.codes))
	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		hc := e.codes[symbol]
		if hc.Size == 0 {
			fmt.Fprintf(&buf, "\tEncode(%d) = nil\n", symbol)
		} else {
			fmt.Fprintf(&buf, "\tEncode(%d) = %s\n", symbol, hc)
		}
	}
	buf.WriteString("}\n")
	return buf.String()
}

// GoString returns a Go expression that would reconstruct this Encoder.
func (e Encoder) GoString() string {
	var buf strings.Builder
	buf.WriteString("NewEncoderFromSizes([]byte{")
	for index, hc := range e.codes {
		if index != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(strconv.FormatUint(uint64(hc.Size), 10))
	}
	buf.WriteString("})")
	return buf.String()
}

// String returns a brief string representation.
func (e Encoder) String() string {
	return fmt.Sprintf(
		"(Huffman encoder with %d symbols, with coded lengths of %d .. %d bits)",
		len(e.codes),
		e.minSize,
		e.maxSize,
	)
}

// MarshalJSON renders this Encoder as JSON data.
func (e Encoder) MarshalJSON() ([]byte, error) {
	length := uint(len(e.codes))
	arr := make([]uint, length)
	for i := uint(0); i < length; i++ {
		arr[i] = uint(e.codes[i].Size)
	}
	return json.Marshal(arr)
}

// UnmarshalJSON initializes this Encoder from JSON data.
func (e *Encoder) UnmarshalJSON(raw []byte) error {
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

	return e.InitFromSizes(sizes)
}

// firstPass computes the "first pass" of Huffman code assignment, which is to
// determine and populate codes[Symbol].Size.  We also compute minSize and
// maxSize while we're here.
//
func firstPass(codes []Code, nodes []symbolAndFreq, minSize *byte, maxSize *byte) {
	nodeLen := uint32(len(nodes))
	nodeLog := log2uint32(nodeLen)

	// Step 1: build a minheap.

	h := freqHeap{nodes}
	h.Init()

	// Step 2: process the minheap by popping two symbols, combining them
	// into a new synthetic symbol, and pushing the new symbol back onto
	// the minheap.
	//
	// Synthetic symbols are distinguished from natural symbols by their
	// sign: "natural" symbols are zero- or positive-valued, while
	// "synthetic" symbols are negative-valued.  math.MinInt32 is the 0'th
	// synthetic symbol, and the subsequent ones are assigned as
	// consecutive integers approaching 0 from below.
	//
	// We probably need only log2(len(nodes)) synthetic symbols.

	type syntheticSymbol struct {
		left  Symbol
		right Symbol
	}

	syntheticSymbols := make([]syntheticSymbol, 0, nodeLog)
	nextSyntheticSymbol := Symbol(math.MinInt32)

	for h.Len() > 1 {
		a := heap.Pop(&h).(symbolAndFreq)
		b := heap.Pop(&h).(symbolAndFreq)

		// Compute freqSum using saturating addition
		freqSum := a.freq + b.freq
		if freqSum < a.freq {
			freqSum = math.MaxUint32
		}

		syntheticSymbols = append(syntheticSymbols, syntheticSymbol{a.symbol, b.symbol})
		heap.Push(&h, symbolAndFreq{nextSyntheticSymbol, freqSum})
		nextSyntheticSymbol++
	}

	// root is the root of our tree.  This is not the *actual* Huffman code
	// tree that we'll be using, because it's not necessarily canonical,
	// but it's good enough to tell us the bit length for each natural
	// symbol's canonical code.
	root := heap.Pop(&h).(symbolAndFreq)

	// Step 3: use a stack to walk the tree.
	//
	// The current stack depth tells us how many bits are in the Huffman
	// code represented by this tree, which is also equal to the number of
	// bits in the canonical Huffman code.  As with the syntheticSymbols
	// array, the maximum stack depth should be about log2(len(nodes));
	// natural symbols never get pushed onto the stack, only synthetic
	// ones.
	//
	// We use stackItem.x to keep track of where we are in the tree walk:
	//   x=0 → We just arrived at stackItem for the first time
	//   x=1 → We have already processed the left child
	//   x=2 → We have already processed both children
	//
	// First we define the needed stack operations as closures, and then
	// the final tree-walking loop will be fairly trivial.

	type stackItem struct {
		s Symbol
		x byte
	}

	stack := make([]stackItem, 0, nodeLog)
	var stackLen uint
	var hasMinMax bool

	stackTop := func() *stackItem {
		return &stack[stackLen-1]
	}

	stackPush := func(symbol Symbol) {
		stack = append(stack, stackItem{s: symbol, x: 0})
		stackLen++
	}

	stackPop := func() {
		stackLen--
		stack[stackLen] = stackItem{}
		stack = stack[:stackLen]
	}

	leftChild := func(item *stackItem) Symbol {
		index := int32(item.s) - math.MinInt32
		return syntheticSymbols[index].left
	}

	rightChild := func(item *stackItem) Symbol {
		index := int32(item.s) - math.MinInt32
		return syntheticSymbols[index].right
	}

	processChild := func(child Symbol) {
		if child < 0 {
			stackPush(child)
			return
		}

		size := byte(stackLen)
		codes[child].Size = size
		if !hasMinMax {
			hasMinMax = true
			*minSize = size
			*maxSize = size
		} else if *minSize > size {
			*minSize = size
		} else if *maxSize < size {
			*maxSize = size
		}
	}

	// And now the tree-walking loop.
	stackPush(root.symbol)
	for stackLen != 0 {
		top := stackTop()
		x := top.x
		top.x++
		switch x {
		case 0:
			processChild(leftChild(top))
		case 1:
			processChild(rightChild(top))
		case 2:
			stackPop()
		}
	}
}

// secondPass computes the "second pass" of Huffman code assignment, which
// involves transforming the (Symbol, codes[Symbol].Size) assignments from
// phase one into a canonical Huffman code written back to codes[Symbol].Bits.
func secondPass(codes []Code) error {
	// Step 1: sort the symbols by (codes[Symbol].Size, Symbol) ascending.

	numSymbols := Symbol(len(codes))
	sorted := make(bySize, 0, numSymbols)
	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		size := codes[symbol].Size
		if size == 0 {
			continue
		}

		// forbid codes with sizes greater than maxBitsPerCode
		if size > maxBitsPerCode {
			return fmt.Errorf("invalid bit length while constructing Huffman tree: got %d, max %d", size, maxBitsPerCode)
		}

		sorted = append(sorted, symbolAndSize{symbol, size})
	}
	sorted.Sort()

	// Step 2: assign the codes sequentially, per the algorithm detailed at
	// <https://en.wikipedia.org/w/index.php?title=Canonical_Huffman_code&oldid=999983137>.

	lastSize := sorted[0].size
	nextCode := uint32(0)
	for _, item := range sorted {
		if item.size > lastSize {
			nextCode <<= (item.size - lastSize)
			lastSize = item.size
		}

		mask := (uint32(1) << item.size) - 1
		if (nextCode &^ mask) != 0 {
			return fmt.Errorf("too many symbols have a code length of %d", item.size)
		}

		codes[item.symbol].Bits = reverseBits(item.size, nextCode)
		nextCode++
	}
	return nil
}

// type symbolAndFreq + type freqHeap {{{

type symbolAndFreq struct {
	symbol Symbol
	freq   uint32
}

type freqHeap struct {
	list []symbolAndFreq
}

func (h *freqHeap) Init() {
	heap.Init(h)
}

func (h *freqHeap) Fix(i int) {
	heap.Fix(h, i)
}

func (h *freqHeap) Len() int {
	return len(h.list)
}

func (h *freqHeap) Swap(i, j int) {
	h.list[i], h.list[j] = h.list[j], h.list[i]
}

func (h *freqHeap) Less(i, j int) bool {
	a, b := h.list[i], h.list[j]
	if a.freq != b.freq {
		return a.freq < b.freq
	}
	return uint32(a.symbol) < uint32(b.symbol)
}

func (h *freqHeap) Push(x interface{}) {
	h.list = append(h.list, x.(symbolAndFreq))
}

func (h *freqHeap) Pop() interface{} {
	last := uint(len(h.list)) - 1
	x := h.list[last]
	h.list = h.list[:last]
	return x
}

var _ heap.Interface = (*freqHeap)(nil)

// }}}

// type symbolAndSize + type bySize {{{

type symbolAndSize struct {
	symbol Symbol
	size   byte
}

type bySize []symbolAndSize

func (list bySize) Len() int {
	return len(list)
}

func (list bySize) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list bySize) Less(i, j int) bool {
	a, b := list[i], list[j]
	ay, ai := a.symbol, a.size
	by, bi := b.symbol, b.size
	if ai != bi {
		return ai < bi
	}
	return ay < by
}

func (list bySize) Sort() {
	sort.Sort(list)
}

var _ sort.Interface = bySize(nil)

// }}}
