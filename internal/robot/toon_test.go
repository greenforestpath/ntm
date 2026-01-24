package robot

import (
	"testing"
	"time"
)

// =============================================================================
// TOON Encoder Unit Tests
// =============================================================================

func TestToonEncode_Primitives(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"nil", nil},
		{"bool true", true},
		{"bool false", false},
		{"int", 42},
		{"negative int", -123},
		{"uint", uint(100)},
		{"float", 3.14159},
		{"float no trailing zeros", 1.5},
		{"float whole number", 2.0},
		{"string simple", "hello"},
		{"string with spaces", "hello world"},
		{"string with special chars", "hello\nworld"},
		{"string empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertToonRoundTrip(t, tc.input)
		})
	}
}

func TestToonEncode_SimpleArrays(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		assertToonRoundTrip(t, []int{})
	})

	t.Run("int slice", func(t *testing.T) {
		assertToonRoundTrip(t, []int{1, 2, 3})
	})

	t.Run("string slice", func(t *testing.T) {
		assertToonRoundTrip(t, []string{"a", "b", "c"})
	})
}

func TestToonEncode_TabularArrays(t *testing.T) {
	t.Run("uniform maps", func(t *testing.T) {
		input := []map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}
		assertToonRoundTrip(t, input)
	})

	t.Run("uniform structs", func(t *testing.T) {
		type Person struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		input := []Person{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}
		assertToonRoundTrip(t, input)
	})
}

func TestToonEncode_Objects(t *testing.T) {
	t.Run("simple map", func(t *testing.T) {
		input := map[string]int{"count": 42, "value": 100}
		assertToonRoundTrip(t, input)
	})

	t.Run("simple struct", func(t *testing.T) {
		type Config struct {
			Port    int    `json:"port"`
			Host    string `json:"host"`
			Enabled bool   `json:"enabled"`
		}
		input := Config{Port: 8080, Host: "localhost", Enabled: true}
		assertToonRoundTrip(t, input)
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]int{}
		assertToonRoundTrip(t, input)
	})
}

func TestToonEncode_TabSafetyRoundTrip(t *testing.T) {
	input := []map[string]string{
		{"name": "Alice", "desc": "has\ttab"},
		{"name": "Bob", "desc": "normal"},
	}
	assertToonRoundTrip(t, input)
}

func TestToonEncode_NestedRoundTrip(t *testing.T) {
	input := []map[string]interface{}{
		{"id": 1, "tags": []string{"a", "b"}},
		{"id": 2, "tags": []string{"c"}},
	}
	assertToonRoundTrip(t, input)
}

func TestToonEncode_PointerHandling(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var ptr *int
		assertToonRoundTrip(t, ptr)
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		val := 42
		ptr := &val
		assertToonRoundTrip(t, ptr)
	})
}

func TestToonEncode_TimeHandling(t *testing.T) {
	input := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	assertToonRoundTrip(t, input)
}

func TestToonEncode_JSONTagHandling(t *testing.T) {
	type Item struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		internal string // unexported, should be skipped
		Ignored  string `json:"-"` // explicitly ignored
		OmitZero int    `json:"omit_zero,omitempty"`
	}

	input := Item{ID: 1, Name: "test", internal: "secret", Ignored: "skip", OmitZero: 0}
	assertToonRoundTrip(t, input)
}

func TestToonEncode_RobotPayloads(t *testing.T) {
	t.Run("RobotResponse", func(t *testing.T) {
		resp := NewRobotResponse(true)
		assertToonRoundTrip(t, resp)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		resp := NewErrorResponse(nil, ErrCodeInternalError, "test hint")
		assertToonRoundTrip(t, resp)
	})
}

func TestToonEncode_JSONMarshalError(t *testing.T) {
	ch := make(chan int)
	if _, err := toonEncode(ch, "\t"); err == nil {
		t.Fatal("expected json marshal error, got nil")
	}
}
