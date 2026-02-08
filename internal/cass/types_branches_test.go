package cass

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// FlexTime.UnmarshalJSON — missing float input branch (86.2% → higher)
// ---------------------------------------------------------------------------

func TestFlexTimeUnmarshalJSON_Float(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantSec int64
	}{
		{"fractional seconds", `1702200000.5`, 1702200000},
		{"high precision", `1702200000.123`, 1702200000},
		{"zero fractional", `1702200000.0`, 1702200000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ft FlexTime
			err := ft.UnmarshalJSON([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ft.Time.Unix() != tc.wantSec {
				t.Errorf("got Unix=%d, want %d", ft.Time.Unix(), tc.wantSec)
			}
		})
	}
}

func TestFlexTimeUnmarshalJSON_FloatNanoseconds(t *testing.T) {
	t.Parallel()

	var ft FlexTime
	err := ft.UnmarshalJSON([]byte(`1702200000.5`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 0.5 seconds = 500_000_000 nanoseconds
	nsec := ft.Time.Nanosecond()
	if nsec < 499000000 || nsec > 501000000 {
		t.Errorf("expected ~500000000 nanoseconds, got %d", nsec)
	}
}

func TestFlexTimeUnmarshalJSON_UnknownFormat(t *testing.T) {
	t.Parallel()

	var ft FlexTime
	err := ft.UnmarshalJSON([]byte(`{"nested": true}`))
	if err == nil {
		t.Error("expected error for object input")
	}
}

func TestFlexTimeUnmarshalJSON_ArrayFormat(t *testing.T) {
	t.Parallel()

	var ft FlexTime
	err := ft.UnmarshalJSON([]byte(`[1,2,3]`))
	if err == nil {
		t.Error("expected error for array input")
	}
}

func TestFlexTimeUnmarshalJSON_NullIsZero(t *testing.T) {
	t.Parallel()

	var ft FlexTime
	ft.Time = time.Now() // Set to non-zero first
	err := ft.UnmarshalJSON([]byte(`null`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ft.Time.IsZero() {
		t.Errorf("expected zero time for null, got %v", ft.Time)
	}
}
