package bitset

import (
	"math/bits"
)

const UINT64S = 37
const BYTES = UINT64S * 8
const BITS = BYTES * 8

// There is a bit for each word
type BitSet [UINT64S]uint64

// the wordSize of a bit set
const wordSize = 64

// wordMask is wordSize-1, used for bit indexing in a word
const wordMask = wordSize - 1

// log2WordSize is lg(wordSize)
const log2WordSize = 6

// wordsIndex calculates the index of words in a `uint64`
func wordsIndex(i uint) uint {
	return i & wordMask
}

const m0 = 0x5555555555555555 // 01010101 ...
const m1 = 0x3333333333333333 // 00110011 ...
const m2 = 0x0f0f0f0f0f0f0f0f // 00001111 ...

const uintSize = 32 << (^uint(0) >> 63) // 32 or 64

// UintSize is the size of a uint in bits.
const UintSize = uintSize

// allBits has every bit set
const allBits uint64 = 0xffffffffffffffff

// OnesCount64 returns the number of one bits ("population count") in x.
func OnesCount64(x uint64) int {
	// Implementation: Parallel summing of adjacent bits.
	// See "Hacker's Delight", Chap. 5: Counting Bits.
	// The following pattern shows the general approach:
	//
	//   x = x>>1&(m0&m) + x&(m0&m)
	//   x = x>>2&(m1&m) + x&(m1&m)
	//   x = x>>4&(m2&m) + x&(m2&m)
	//   x = x>>8&(m3&m) + x&(m3&m)
	//   x = x>>16&(m4&m) + x&(m4&m)
	//   x = x>>32&(m5&m) + x&(m5&m)
	//   return int(x)
	//
	// Masking (& operations) can be left away when there's no
	// danger that a field's sum will carry over into the next
	// field: Since the result cannot be > 64, 8 bits is enough
	// and we can ignore the masks for the shifts by 8 and up.
	// Per "Hacker's Delight", the first line can be simplified
	// more, but it saves at best one instruction, so we leave
	// it alone for clarity.
	const m = 1<<64 - 1
	x = x>>1&(m0&m) + x&(m0&m)
	x = x>>2&(m1&m) + x&(m1&m)
	x = (x>>4 + x) & (m2 & m)
	x += x >> 8
	x += x >> 16
	x += x >> 32
	return int(x) & (1<<7 - 1)
}

// Is the length an exact multiple of word sizes?
func (b *BitSet) isLenExactMultiple(length uint) bool {
	return wordsIndex(length) == 0
}

func popcntSlice(s *BitSet) (cnt uint64) {
	for _, x := range s {
		if x == 0 {
			continue
		}
		cnt += uint64(OnesCount64(x))
	}
	return
}

func New(bits uint) *BitSet {
	if bits > BITS {
		panic("Do not support that many bits")
	}
	return &BitSet{}
}

func (b *BitSet) Set(i uint) *BitSet {
	b[i>>log2WordSize] |= 1 << wordsIndex(i)
	return b
}

// Intersection of base set and other set
// This is the BitSet equivalent of & (and)
// In case of allocation failure, the function will return an empty BitSet.
func (b *BitSet) Intersection(compare *BitSet) *BitSet {
	var result BitSet
	for i, word := range b {
		result[i] = word & compare[i]
	}
	return &result
}

func (b *BitSet) IntersectionInPlace(compare *BitSet, result *BitSet) {
	for i, word := range b {
		result[i] = word & compare[i]
	}
}

// Intersection of base set and other set.  Assume compare has more zero words than b.
// looking for the total number of bits set after the and operation.
func (b *BitSet) IntersectionBitCount(compare *BitSet) int {
	cnt := 0
	for i, bWord := range b {
		if bWord == 0 {
			continue
		}
		cWord := compare[i]
		if cWord == 0 { // this test might cost more than it's worth
			continue
		}
		result := bWord & cWord
		if result == 0 {
			continue
		}
		cnt += OnesCount64(result)
	}
	return cnt
}

// Difference of base set and other set
// This is the BitSet equivalent of &^ (and not)
func (b *BitSet) Difference(compare *BitSet) *BitSet {
	result := &BitSet{}
	for i := range compare {
		result[i] = b[i] ^ compare[i]
	}
	return result
}

// Count (number of set bits).
// Also known as "popcount" or "population count".
func (b *BitSet) Count() uint {
	return uint(popcntSlice(b))
}

// Clean last word by setting unused bits to 0
func (b *BitSet) cleanLastWord(length uint) {
	wi := wordsIndex(length)
	b[length/wordSize] &= allBits >> (wordSize - wi)
	// b[len(b)-1] &= allBits >> (wordSize - wordsIndex(length))
}

// SetAll sets the entire BitSet
func (b *BitSet) SetAll(length uint) *BitSet {
	for i := range (length / wordSize) + 1 {
		b[i] = allBits
	}

	b.cleanLastWord(length)
	return b
}

// NextSet returns the next bit set from the specified index,
// including possibly the current index
// along with an error code (true = valid, false = no set bit found)
// for i,e := v.NextSet(0); e; i,e = v.NextSet(i + 1) {...}
//
// Users concerned with performance may want to use NextSetMany to
// retrieve several values at once.
func (b *BitSet) NextSet(i uint) (uint, bool) {
	x := int(i >> log2WordSize)
	if x >= len(b) {
		return 0, false
	}

	// process first (partial) word
	word := b[x] >> wordsIndex(i)
	if word != 0 {
		return i + uint(bits.TrailingZeros64(word)), true
	}

	// process the following full words until next bit is set
	// x < len(b.set), no out-of-bounds panic in following slice expression
	x++
	for idx, word := range b[x:] {
		if word != 0 {
			return uint((x+idx)<<log2WordSize + bits.TrailingZeros64(word)), true
		}
	}

	return 0, false
}
