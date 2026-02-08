package output

import "testing"

// ---------------------------------------------------------------------------
// Truncate — cover maxLen <= 0 and large first rune branches (85% → 100%)
// ---------------------------------------------------------------------------

func TestTruncate_ZeroMaxLen(t *testing.T) {
	t.Parallel()

	got := Truncate("hello", 0)
	if got != "" {
		t.Errorf("Truncate(\"hello\", 0) = %q, want \"\"", got)
	}
}

func TestTruncate_NegativeMaxLen(t *testing.T) {
	t.Parallel()

	got := Truncate("hello", -5)
	if got != "" {
		t.Errorf("Truncate(\"hello\", -5) = %q, want \"\"", got)
	}
}

func TestTruncate_LargeFirstRuneSmallMaxLen(t *testing.T) {
	t.Parallel()

	// Emoji is 4 bytes, maxLen=2 means first rune is larger than maxLen.
	// The loop finds no valid boundary (lastValid=0), returns "".
	got := Truncate("🎉hello", 2)
	if got != "" {
		t.Errorf("Truncate(\"🎉hello\", 2) = %q, want \"\"", got)
	}
}

func TestTruncate_ExactlyThreeCharsMaxLen(t *testing.T) {
	t.Parallel()

	// Edge case: maxLen=3 with longer string, should return first 3 chars (no "...")
	got := Truncate("abcdef", 3)
	if got != "abc" {
		t.Errorf("Truncate(\"abcdef\", 3) = %q, want \"abc\"", got)
	}
}

func TestTruncate_MultibyteBoundary(t *testing.T) {
	t.Parallel()

	// "日本語" is 9 bytes (3 bytes each), maxLen=5 should find boundary at 3
	got := Truncate("日本語", 5)
	// With maxLen=5 and maxLen<=3 false, targetLen=2, first rune at 0 (<=2),
	// second at 3 (>2), so return s[:0]+"..." = "..."
	if got != "..." {
		t.Errorf("Truncate(\"日本語\", 5) = %q, want \"...\"", got)
	}
}

func TestTruncate_FinalReturnBranch(t *testing.T) {
	t.Parallel()

	// Test where the loop completes without early return.
	// Need: len(s) > maxLen, maxLen > 3, and all rune starts <= targetLen.
	// "ab🎉" is 6 bytes (2+4), maxLen=5, targetLen=2
	// Runes at 0 (a), 1 (b), 2 (🎉 4-byte)
	// Loop: i=0 <= 2 ✓, i=1 <= 2 ✓, i=2 <= 2 ✓, loop ends (next would be i=6 but string ends).
	// prevI=2, return s[:2]+"..." = "ab..."
	got := Truncate("ab🎉", 5)
	if got != "ab..." {
		t.Errorf("Truncate(\"ab🎉\", 5) = %q, want \"ab...\"", got)
	}
}
