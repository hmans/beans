package bean

// Fractional indexing generates order keys that sort lexicographically.
// Keys are strings of base-62 digits (0-9, A-Z, a-z).
// Given any two keys, a new key can always be generated between them.
//
// This is used for manual ordering of beans: moving a bean only requires
// updating that one bean's order key, not renumbering neighbors.

const base62Digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// OrderBetween returns a key that sorts lexicographically between a and b.
// If a is "", it generates a key before b.
// If b is "", it generates a key after a.
// If both are "", it returns the midpoint "V" (roughly middle of base-62).
func OrderBetween(a, b string) string {
	if a == "" && b == "" {
		return "V"
	}
	if a == "" {
		return decrementKey(b)
	}
	if b == "" {
		return incrementKey(a)
	}
	return midpoint(a, b)
}

// midpoint computes a key lexicographically between a and b.
// Precondition: a < b.
func midpoint(a, b string) string {
	// Pad to equal length
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	padA := padRight(a, maxLen)
	padB := padRight(b, maxLen)

	// Convert to digit indices
	digitsA := toIndices(padA)
	digitsB := toIndices(padB)

	// Try to find a midpoint at each position
	for i := 0; i < maxLen; i++ {
		if digitsA[i] < digitsB[i]-1 {
			// There's room between these digits
			mid := (digitsA[i] + digitsB[i]) / 2
			result := make([]int, i+1)
			copy(result, digitsA[:i])
			result[i] = mid
			return fromIndices(result)
		}
		if digitsA[i] == digitsB[i] {
			continue
		}
		// digitsA[i] == digitsB[i]-1: need to go deeper
		// Take a's prefix up to i, then find midpoint between a's suffix and max
		result := make([]int, i+1)
		copy(result, digitsA[:i+1])
		// Append a digit halfway between a's next digit (or 0) and max (61)
		nextA := 0
		if i+1 < len(digitsA) {
			nextA = digitsA[i+1]
		}
		mid := (nextA + 62) / 2
		result = append(result, mid)
		return fromIndices(result)
	}

	// a and b are equal up to maxLen; extend with midpoint
	result := make([]int, maxLen+1)
	copy(result, digitsA)
	result[maxLen] = 31 // middle of 0-61
	return fromIndices(result)
}

// incrementKey generates a key after the given key.
func incrementKey(key string) string {
	digits := toIndices(key)

	// Try to increment the last digit
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < 61 {
			result := make([]int, i+1)
			copy(result, digits[:i])
			result[i] = (digits[i] + 62) / 2 // midpoint between current and max
			if result[i] == digits[i] {
				result[i] = digits[i] + 1
			}
			return fromIndices(result)
		}
	}

	// All digits are max; append a midpoint
	result := make([]int, len(digits)+1)
	copy(result, digits)
	result[len(digits)] = 31
	return fromIndices(result)
}

// decrementKey generates a key before the given key.
func decrementKey(key string) string {
	digits := toIndices(key)

	// Find the rightmost non-zero digit
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] > 1 {
			// Halve this digit (midpoint between 0 and current)
			result := make([]int, i+1)
			copy(result, digits[:i])
			result[i] = digits[i] / 2
			return fromIndices(result)
		}
		if digits[i] == 1 {
			// Can't halve to 0 and truncate (would shorten the key and might not be less).
			// Instead, keep prefix up to i as 0, and append a high digit.
			result := make([]int, i+2)
			copy(result, digits[:i])
			result[i] = 0
			result[i+1] = 31 // midpoint
			return fromIndices(result)
		}
	}

	// All digits are 0 — extend with a midpoint to create something that sorts before.
	// "0" + "V" < "0" + anything... but "0V" > "0" lexicographically.
	// We need to go shorter. Use a single "0" padded approach:
	// Actually "0" is already the smallest single-char key. For "00", "0" < "00" lex.
	// So just return "0" if key is longer than 1 char and all zeros.
	if len(key) > 1 {
		return key[:len(key)-1]
	}
	// key is "0" — the absolute minimum. Shouldn't happen in practice.
	return "0"
}

func padRight(s string, length int) string {
	for len(s) < length {
		s += string(base62Digits[0])
	}
	return s
}

func toIndices(s string) []int {
	result := make([]int, len(s))
	for i, ch := range s {
		result[i] = indexOf(byte(ch))
	}
	return result
}

func fromIndices(indices []int) string {
	result := make([]byte, len(indices))
	for i, idx := range indices {
		if idx < 0 {
			idx = 0
		}
		if idx > 61 {
			idx = 61
		}
		result[i] = base62Digits[idx]
	}
	return string(result)
}

func indexOf(ch byte) int {
	for i := 0; i < len(base62Digits); i++ {
		if base62Digits[i] == ch {
			return i
		}
	}
	return 0
}
