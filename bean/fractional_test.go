package bean

import (
	"testing"
)

func TestOrderBetween_BothEmpty(t *testing.T) {
	result := OrderBetween("", "")
	if result != "V" {
		t.Errorf("OrderBetween(\"\", \"\") = %q, want %q", result, "V")
	}
}

func TestOrderBetween_BeforeKey(t *testing.T) {
	result := OrderBetween("", "V")
	if result >= "V" {
		t.Errorf("OrderBetween(\"\", \"V\") = %q, should be < \"V\"", result)
	}
	if result == "" {
		t.Error("OrderBetween(\"\", \"V\") should not be empty")
	}
}

func TestOrderBetween_AfterKey(t *testing.T) {
	result := OrderBetween("V", "")
	if result <= "V" {
		t.Errorf("OrderBetween(\"V\", \"\") = %q, should be > \"V\"", result)
	}
}

func TestOrderBetween_Between(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"A", "Z"},
		{"V", "X"},
		{"a", "z"},
		{"0", "z"},
		{"Va", "Vz"},
		{"V", "W"},
	}

	for _, tt := range tests {
		result := OrderBetween(tt.a, tt.b)
		if result <= tt.a || result >= tt.b {
			t.Errorf("OrderBetween(%q, %q) = %q, should be between", tt.a, tt.b, result)
		}
	}
}

func TestOrderBetween_AdjacentDigits(t *testing.T) {
	// When digits are adjacent (e.g., A and B), we need to go deeper
	result := OrderBetween("A", "B")
	if result <= "A" || result >= "B" {
		t.Errorf("OrderBetween(\"A\", \"B\") = %q, should be between", result)
	}
}

func TestOrderBetween_ManyInsertions(t *testing.T) {
	// Simulate inserting many items at the end
	keys := []string{"V"}
	for i := 0; i < 50; i++ {
		newKey := OrderBetween(keys[len(keys)-1], "")
		if newKey <= keys[len(keys)-1] {
			t.Fatalf("insertion %d: %q should be > %q", i, newKey, keys[len(keys)-1])
		}
		keys = append(keys, newKey)
	}

	// Verify all are in order
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Errorf("keys not in order at %d: %q <= %q", i, keys[i], keys[i-1])
		}
	}
}

func TestOrderBetween_ManyInsertionsAtStart(t *testing.T) {
	keys := []string{"V"}
	for i := 0; i < 50; i++ {
		newKey := OrderBetween("", keys[0])
		if newKey >= keys[0] {
			t.Fatalf("insertion %d: %q should be < %q", i, newKey, keys[0])
		}
		keys = append([]string{newKey}, keys...)
	}

	// Verify all are in order
	for i := 1; i < len(keys); i++ {
		if keys[i] <= keys[i-1] {
			t.Errorf("keys not in order at %d: %q <= %q", i, keys[i], keys[i-1])
		}
	}
}

func TestOrderBetween_ManyInsertionsBetween(t *testing.T) {
	// Repeatedly insert between two keys
	a := "A"
	b := "z"
	for i := 0; i < 100; i++ {
		mid := OrderBetween(a, b)
		if mid <= a || mid >= b {
			t.Fatalf("insertion %d: %q should be between %q and %q", i, mid, a, b)
		}
		// Alternate inserting before and after the midpoint
		if i%2 == 0 {
			b = mid
		} else {
			a = mid
		}
	}
}
