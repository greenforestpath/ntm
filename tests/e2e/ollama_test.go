package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/tests/testutil"
)

type ollamaModelsListResponse struct {
	Host       string `json:"host"`
	ModelCount int    `json:"model_count"`
	Models     []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type ollamaSpawnResponse struct {
	Session string `json:"session"`
	Created bool   `json:"created"`
	Panes   []struct {
		Index int    `json:"index"`
		Type  string `json:"type"`
	} `json:"panes"`
}

type fakeOllamaServer struct {
	server *httptest.Server

	mu     sync.Mutex
	models map[string]struct{}
}

func newFakeOllamaServer(t *testing.T, initialModels []string, tagsDelay time.Duration) *fakeOllamaServer {
	t.Helper()

	s := &fakeOllamaServer{
		models: make(map[string]struct{}),
	}
	for _, m := range initialModels {
		if strings.TrimSpace(m) == "" {
			continue
		}
		s.models[strings.TrimSpace(m)] = struct{}{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		if tagsDelay > 0 {
			time.Sleep(tagsDelay)
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		models := make([]map[string]any, 0, len(s.models))
		for name := range s.models {
			models = append(models, map[string]any{
				"name":        name,
				"size":        int64(1024),
				"digest":      "sha256:test",
				"modified_at": time.Now().UTC().Format(time.RFC3339),
				"details": map[string]any{
					"family":         "llama",
					"parameter_size": "7B",
				},
			})
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
	})

	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		model := strings.TrimSpace(req.Name)
		if model == "" {
			model = "unknown:model"
		}

		s.mu.Lock()
		s.models[model] = struct{}{}
		s.mu.Unlock()

		flusher, _ := w.(http.Flusher)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "pulling manifest"})
		if flusher != nil {
			flusher.Flush()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "done": true})
		if flusher != nil {
			flusher.Flush()
		}
	})

	s.server = httptest.NewServer(mux)
	t.Cleanup(s.server.Close)
	return s
}

func requireOllamaE2E(t *testing.T, needsTmux bool) {
	t.Helper()

	testutil.RequireE2E(t)
	if needsTmux {
		testutil.RequireTmuxThrottled(t)
	}
	testutil.RequireNTMBinary(t)
	testutil.RequireEnv(t, "OLLAMA_E2E")
}

func TestOllamaE2E_ConnectionChecks(t *testing.T) {
	requireOllamaE2E(t, false)

	logger := testutil.NewTestLogger(t, t.TempDir())
	modelName := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if modelName == "" {
		modelName = "llama2:latest"
	}

	fake := newFakeOllamaServer(t, []string{modelName}, 0)
	host := fake.server.URL
	hostNoScheme := strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")

	logger.LogSection("[E2E-OLLAMA] valid endpoint")
	out := testutil.AssertCommandSuccess(t, logger, "ntm", "--json", "models", "list", "--host", host)
	var listResp ollamaModelsListResponse
	if err := json.Unmarshal(out, &listResp); err != nil {
		t.Fatalf("failed to parse models list output: %v\nraw=%s", err, string(out))
	}
	if listResp.ModelCount < 1 {
		t.Fatalf("expected at least 1 model, got %d", listResp.ModelCount)
	}

	logger.LogSection("[E2E-OLLAMA] custom port/host syntax")
	out = testutil.AssertCommandSuccess(t, logger, "ntm", "--json", "models", "list", "--host", hostNoScheme)
	if err := json.Unmarshal(out, &listResp); err != nil {
		t.Fatalf("failed to parse custom host output: %v\nraw=%s", err, string(out))
	}

	logger.LogSection("[E2E-OLLAMA] invalid host returns connection error")
	out = testutil.AssertCommandFails(t, logger, "ntm", "--json", "models", "list", "--host", "http://127.0.0.1:1")
	outStr := strings.ToLower(string(out))
	if !strings.Contains(outStr, "failed to connect") && !strings.Contains(outStr, "connection refused") {
		t.Fatalf("expected connection failure output, got: %s", string(out))
	}

	logger.LogSection("[E2E-OLLAMA] timeout on slow endpoint")
	slow := newFakeOllamaServer(t, []string{modelName}, 6*time.Second)
	out = testutil.AssertCommandFails(t, logger, "ntm", "--json", "models", "list", "--host", slow.server.URL)
	outStr = strings.ToLower(string(out))
	if !strings.Contains(outStr, "timeout") && !strings.Contains(outStr, "deadline") {
		t.Fatalf("expected timeout/deadline output, got: %s", string(out))
	}
}

func TestOllamaE2E_PullAndSpawnLocalAgent(t *testing.T) {
	requireOllamaE2E(t, true)

	logger := testutil.NewTestLogger(t, t.TempDir())
	modelName := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if modelName == "" {
		modelName = "llama2:latest"
	}

	fake := newFakeOllamaServer(t, nil, 0)
	host := fake.server.URL

	logger.LogSection("[E2E-OLLAMA] model pull")
	testutil.AssertCommandSuccess(t, logger, "ntm", "--json", "models", "pull", "--host", host, modelName)

	logger.LogSection("[E2E-OLLAMA] model list includes pulled model")
	out := testutil.AssertCommandSuccess(t, logger, "ntm", "--json", "models", "list", "--host", host)
	var listResp ollamaModelsListResponse
	if err := json.Unmarshal(out, &listResp); err != nil {
		t.Fatalf("failed to parse models list output: %v\nraw=%s", err, string(out))
	}
	found := false
	for _, m := range listResp.Models {
		if m.Name == modelName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pulled model %q not found in list response: %+v", modelName, listResp.Models)
	}

	projectsBase := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	config := fmt.Sprintf(`projects_base = %q

[agents]
ollama = "sh -lc 'echo local-ollama-ready {{shellQuote (.Model | default \"codellama:latest\")}}; sleep 2'"
claude = "echo claude"
codex = "echo codex"
gemini = "echo gemini"
`, projectsBase)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	sessionName := fmt.Sprintf("e2e_ollama_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = testutil.SessionExists(sessionName)
		_, _ = logger.Exec("ntm", "--config", configPath, "--json", "kill", sessionName, "--force")
	})

	logger.LogSection("[E2E-OLLAMA] spawn local agent")
	out = testutil.AssertCommandSuccess(
		t,
		logger,
		"ntm", "--config", configPath, "--json",
		"spawn", sessionName,
		"--local=1",
		"--local-model", modelName,
		"--local-host", host,
		"--no-hooks",
	)

	var spawnResp ollamaSpawnResponse
	if err := json.Unmarshal(out, &spawnResp); err != nil {
		t.Fatalf("failed to parse spawn response: %v\nraw=%s", err, string(out))
	}
	if !spawnResp.Created {
		t.Fatalf("expected created=true in spawn response")
	}

	localPane := 0
	for _, p := range spawnResp.Panes {
		if p.Type == "ollama" {
			localPane = p.Index
			break
		}
	}
	if localPane == 0 {
		t.Fatalf("expected an ollama pane in spawn response, got: %+v", spawnResp.Panes)
	}

	time.Sleep(500 * time.Millisecond)
	content, err := testutil.CapturePane(sessionName, localPane)
	if err != nil {
		t.Fatalf("failed to capture pane %d: %v", localPane, err)
	}
	if !strings.Contains(content, "local-ollama-ready") {
		t.Fatalf("expected local ollama marker in pane content, got: %s", content)
	}
}
