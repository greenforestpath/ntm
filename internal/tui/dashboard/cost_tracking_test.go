package dashboard

import "testing"

func TestTailDelta(t *testing.T) {
	tests := []struct {
		name    string
		prev    string
		current string
		want    string
	}{
		{
			name:    "empty prev returns current",
			prev:    "",
			current: "a\nb\n",
			want:    "a\nb",
		},
		{
			name:    "identical returns empty",
			prev:    "a\nb\nc\n",
			current: "a\nb\nc\n",
			want:    "",
		},
		{
			name:    "overlap suffix/prefix returns delta",
			prev:    "A\nB\nC\n",
			current: "B\nC\nD\n",
			want:    "D",
		},
		{
			name:    "no overlap returns full current",
			prev:    "A\nB\nC\n",
			current: "X\nY\nZ\n",
			want:    "X\nY\nZ",
		},
		{
			name:    "longer overlap returns multiple lines",
			prev:    "A\nB\nC\nD\n",
			current: "C\nD\nE\nF\n",
			want:    "E\nF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tailDelta(tt.prev, tt.current)
			if got != tt.want {
				t.Errorf("tailDelta() = %q, want %q", got, tt.want)
			}
		})
	}
}
