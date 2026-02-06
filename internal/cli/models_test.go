package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/agent/ollama"
)

func TestModelsCmd(t *testing.T) {
	cmd := newModelsCmd()
	if cmd.Use != "models" {
		t.Fatalf("expected Use=models, got %q", cmd.Use)
	}

	expectedSubs := map[string]bool{
		"list":      false,
		"pull":      false,
		"remove":    false,
		"recommend": false,
	}
	for _, sub := range cmd.Commands() {
		if _, ok := expectedSubs[sub.Name()]; ok {
			expectedSubs[sub.Name()] = true
		}
	}
	for name, found := range expectedSubs {
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

func TestRecommendationsForVRAM(t *testing.T) {
	tests := []struct {
		name string
		vram float64
		want []string
	}{
		{
			name: "lt_4gb",
			vram: 3.5,
			want: []string{"codellama:7b-instruct", "deepseek-coder:6.7b"},
		},
		{
			name: "between_4_and_8",
			vram: 4.0,
			want: []string{"codellama:13b", "deepseek-coder:6.7b-instruct"},
		},
		{
			name: "between_8_and_16",
			vram: 12.0,
			want: []string{"codellama:34b", "deepseek-coder:33b"},
		},
		{
			name: "gte_16",
			vram: 24.0,
			want: []string{"mixtral:8x7b", "codellama:70b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recommendationsForVRAM(tt.vram)
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d want=%d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d]=%q want=%q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseNvidiaSMIMemoryMB(t *testing.T) {
	raw := "4096\n8192\n"
	got := parseNvidiaSMIMemoryMB(raw)
	if got != 8192 {
		t.Fatalf("expected 8192, got %.0f", got)
	}
}

func TestParseDarwinVRAMGB(t *testing.T) {
	raw := `
Graphics/Displays:

    Chipset Model: Apple M3
    VRAM (Total): 8 GB
    VRAM (Dynamic, Max): 1536 MB
`
	got := parseDarwinVRAMGB(raw)
	if got != 8 {
		t.Fatalf("expected 8, got %.1f", got)
	}
}

func TestSuggestModelCleanup(t *testing.T) {
	now := time.Now()
	models := []ollama.Model{
		{Name: "newer", ModifiedAt: now.Add(-1 * time.Hour), Size: 2},
		{Name: "oldest", ModifiedAt: now.Add(-72 * time.Hour), Size: 1},
		{Name: "middle", ModifiedAt: now.Add(-24 * time.Hour), Size: 3},
	}

	got := suggestModelCleanup(models, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 models, got %d", len(got))
	}
	if got[0] != "oldest" || got[1] != "middle" {
		t.Fatalf("unexpected cleanup order: %#v", got)
	}
}

func TestRunModelsList_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"name":        "codellama:latest",
						"size":        int64(1024),
						"digest":      "sha256:abc",
						"modified_at": time.Now().UTC().Format(time.RFC3339),
						"details": map[string]any{
							"family":         "llama",
							"parameter_size": "7B",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldJSON := jsonOutput
	defer func() { jsonOutput = oldJSON }()
	jsonOutput = true

	if err := runModelsList(server.URL); err != nil {
		t.Fatalf("runModelsList failed: %v", err)
	}
}

func TestRunModelsPull_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
		case "/api/pull":
			flusher, _ := w.(http.Flusher)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "pulling"})
			flusher.Flush()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success"})
			flusher.Flush()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldJSON := jsonOutput
	defer func() { jsonOutput = oldJSON }()
	jsonOutput = true

	if err := runModelsPull(server.URL, "mistral:latest", 5*time.Second); err != nil {
		t.Fatalf("runModelsPull failed: %v", err)
	}
}

func TestRunModelsRemove_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
		case "/api/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldJSON := jsonOutput
	defer func() { jsonOutput = oldJSON }()
	jsonOutput = true

	if err := runModelsRemove(server.URL, "mistral:latest", true); err != nil {
		t.Fatalf("runModelsRemove failed: %v", err)
	}
}

func TestRunModelsPull_InvalidModel(t *testing.T) {
	err := runModelsPull("http://127.0.0.1:11434", "bad model!", time.Second)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "invalid model name") {
		t.Fatalf("unexpected error: %v", err)
	}
}
