package textutil

import "unicode/utf8"

// ByteToRuneMap builds a mapping from byte offset to rune offset for a string.
// byteToRune[bytePos] = runePos. Used when converting regex byte offsets to rune offsets
// for correct Unicode character positioning. Continuation bytes of multi-byte runes
// map to the same rune position as the rune's first byte.
func ByteToRuneMap(s string) []int {
	m := make([]int, len(s)+1)
	r := 0
	for i := 0; i < len(s); {
		m[i] = r
		_, size := utf8.DecodeRuneInString(s[i:])
		// Fill continuation byte positions with the same rune index
		for j := 1; j < size; j++ {
			if i+j < len(s) {
				m[i+j] = r
			}
		}
		i += size
		r++
	}
	m[len(s)] = r
	return m
}
