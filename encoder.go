package huffman

import (
	"bytes"
	"container/heap"
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/chronos-tachyon/assert"
)

// Encoder implements an encoder for canonical Huffman codes.
type Encoder struct {
	codes   []Code
	minSize byte
	maxSize byte
}

// Init initializes this Encoder.  The first argument tells Init how many
// Symbols are in this code's alphabet, and the second argument lists the
// frequency (i.e. number of occurrences) for each Symbol in the code, one for
// each Symbol except that any Symbol not represented in the list is assumed to
// have a frequency of 0.
//
func (e *Encoder) Init(numSymbols int, frequencies []uint32) {
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
		secondPass(codes)
	}

	*e = Encoder{
		codes:   codes,
		minSize: minSize,
		maxSize: maxSize,
	}
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

// Dump writes a programmer-readable debugging dump of the Encoder's current
// state to the given writer.
func (e Encoder) Dump(w io.Writer) (int64, error) {
	var buf bytes.Buffer
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
	return buf.WriteTo(w)
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
func secondPass(codes []Code) {
	// Step 1: sort the symbols by (codes[Symbol].Size, Symbol) ascending.

	numSymbols := Symbol(len(codes))
	sorted := make(bySize, 0, numSymbols)
	for symbol := Symbol(0); symbol < numSymbols; symbol++ {
		hc := codes[symbol]
		if hc.Size == 0 {
			continue
		}
		sorted = append(sorted, symbolAndSize{symbol, hc.Size})
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
		codes[item.symbol].Bits = nextCode
		nextCode++
	}
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
