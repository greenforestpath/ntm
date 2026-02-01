package redaction

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestMain(m *testing.M) {
	// Reset patterns to ensure fresh compilation
	ResetPatterns()
	os.Exit(m.Run())
}

const (
	alphaNum               = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	alphaNumDashUnderscore = alphaNum + "-_"
	upperAlphaNum          = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	awsSecretChars         = alphaNum + "/+="
)

type cachedString struct {
	once  sync.Once
	value string
}

func (c *cachedString) get(gen func() string) string {
	c.once.Do(func() {
		c.value = gen()
	})
	return c.value
}

func randChars(allowed string, n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = allowed[int(b)%len(allowed)]
	}
	return string(out)
}

func openAIMarker() string {
	return base64.StdEncoding.EncodeToString([]byte("OpenAI"))
}

var (
	openAIKey        cachedString
	openAILegacyKey  cachedString
	openAIProjKey    cachedString
	anthropicKey     cachedString
	githubToken      cachedString
	githubFinePAT    cachedString
	githubOAuthToken cachedString
	githubAppToken   cachedString
	awsAccessKey     cachedString
	awsSTSAccessKey  cachedString
	awsSecretKey     cachedString
	googleAPIKey     cachedString
	jwtToken         cachedString
)

// testOpenAIKey constructs a fake OpenAI API key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testOpenAIKey() string {
	// Format: OpenAI API key (prefix + base64 marker + random)
	return openAIKey.get(func() string {
		return "s" + "k-" + randChars(alphaNum, 20) + openAIMarker() + randChars(alphaNum, 24)
	})
}

// testOpenAILegacyKey constructs a fake legacy OpenAI API key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testOpenAILegacyKey() string {
	// Format: legacy OpenAI key (prefix + 48 chars)
	return openAILegacyKey.get(func() string {
		return "s" + "k-" + randChars(alphaNum, 48)
	})
}

// testOpenAIProjKey constructs a fake OpenAI project key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testOpenAIProjKey() string {
	// Format: OpenAI project key (prefix + 40+ chars)
	return openAIProjKey.get(func() string {
		return "s" + "k-" + "proj-" + randChars(alphaNumDashUnderscore, 40)
	})
}

// testAnthropicKey constructs a fake Anthropic API key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testAnthropicKey() string {
	// Format: Anthropic key (prefix + 40+ chars)
	return anthropicKey.get(func() string {
		return "s" + "k-" + "ant-" + randChars(alphaNumDashUnderscore, 40)
	})
}

// testGitHubToken constructs a fake GitHub classic token for testing.
// Generated at runtime to avoid triggering secret scanners.
func testGitHubToken() string {
	// Format: GitHub classic token (prefix + 30+ chars)
	return githubToken.get(func() string {
		return "g" + "hp_" + randChars(alphaNum, 36)
	})
}

// testGitHubFinePAT constructs a fake GitHub fine-grained PAT for testing.
// Generated at runtime to avoid triggering secret scanners.
func testGitHubFinePAT() string {
	// Format: GitHub fine-grained PAT (prefix + 20+ + '_' + 40+)
	return githubFinePAT.get(func() string {
		return "github_" + "pat_" + randChars(alphaNum, 20) + "_" + randChars(alphaNum, 40)
	})
}

// testGitHubOAuthToken constructs a fake GitHub OAuth token for testing.
// Generated at runtime to avoid triggering secret scanners.
func testGitHubOAuthToken() string {
	// Format: GitHub OAuth token (prefix + 30+ chars)
	return githubOAuthToken.get(func() string {
		return "g" + "ho_" + randChars(alphaNum, 36)
	})
}

// testGitHubAppToken constructs a fake GitHub App installation token for testing.
// Generated at runtime to avoid triggering secret scanners.
func testGitHubAppToken() string {
	// Format: GitHub App installation token (prefix + 30+ chars)
	return githubAppToken.get(func() string {
		return "g" + "hs_" + randChars(alphaNum, 36)
	})
}

// testAWSAccessKey constructs a fake AWS access key ID for testing.
// Generated at runtime to avoid triggering secret scanners.
func testAWSAccessKey() string {
	// Format: AWS access key id (prefix + 16)
	return awsAccessKey.get(func() string {
		return "AK" + "IA" + randChars(upperAlphaNum, 16)
	})
}

// testAWSSTSAccessKey constructs a fake AWS STS access key ID for testing.
// Generated at runtime to avoid triggering secret scanners.
func testAWSSTSAccessKey() string {
	// Format: AWS STS access key id (prefix + 16)
	return awsSTSAccessKey.get(func() string {
		return "AS" + "IA" + randChars(upperAlphaNum, 16)
	})
}

// testAWSSecretKey constructs a fake AWS secret access key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testAWSSecretKey() string {
	// 40 chars (matches [a-zA-Z0-9/+=]{40}).
	return awsSecretKey.get(func() string {
		return randChars(awsSecretChars, 40)
	})
}

// testGoogleAPIKey constructs a fake Google API key for testing.
// Generated at runtime to avoid triggering secret scanners.
func testGoogleAPIKey() string {
	// Format: Google API key (prefix + 35)
	return googleAPIKey.get(func() string {
		return "AI" + "za" + randChars(alphaNumDashUnderscore, 35)
	})
}

// testJWT constructs a fake JWT for testing (3 base64url-ish parts).
func testJWT() string {
	return jwtToken.get(func() string {
		return "eyJ" + randChars(alphaNumDashUnderscore, 16) + "." + "eyJ" + randChars(alphaNumDashUnderscore, 16) + "." + randChars(alphaNumDashUnderscore, 32)
	})
}

// testRSAPrivateKeyBlock constructs a fake RSA private key PEM block.
// Split into parts to avoid triggering secret scanners.
func testRSAPrivateKeyBlock() string {
	return "-----BEGIN " + "RSA " + "PRIVATE KEY-----\\n" +
		"MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGyLXJ8B+l0DGKx7mN0wbP6zXuF9S4xGz\\n" +
		"-----END " + "RSA " + "PRIVATE KEY-----"
}

// testOpenSSHPrivateKeyBlock constructs a fake OpenSSH private key block.
// Split into parts to avoid triggering secret scanners.
func testOpenSSHPrivateKeyBlock() string {
	return "-----BEGIN " + "OPENSSH " + "PRIVATE KEY-----\\n" +
		"b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAB\\n" +
		"-----END " + "OPENSSH " + "PRIVATE KEY-----"
}

// testGenericPrivateKeyBlock constructs a fake generic private key PEM block.
// Split into parts to avoid triggering secret scanners.
func testGenericPrivateKeyBlock() string {
	return "-----BEGIN " + "PRIVATE KEY-----\\n" +
		"MIIEvQIBADANBgkqhkiG9w0BAQEFAASC\\n" +
		"-----END " + "PRIVATE KEY-----"
}

// TestFixtures holds the test fixture data structure
type TestFixtures struct {
	Version       string `json:"version"`
	TruePositives []struct {
		Input            string `json:"input"`
		ExpectedCategory string `json:"expected_category"`
		Description      string `json:"description"`
	} `json:"true_positives"`
	TrueNegatives []struct {
		Input       string `json:"input"`
		Description string `json:"description"`
	} `json:"true_negatives"`
	EdgeCases []struct {
		Input              string   `json:"input"`
		ExpectedCategory   string   `json:"expected_category,omitempty"`
		ExpectedCategories []string `json:"expected_categories,omitempty"`
		Description        string   `json:"description"`
	} `json:"edge_cases"`
}

func loadFixtures(t *testing.T) *TestFixtures {
	t.Helper()

	// Look for fixtures in testdata directory
	path := filepath.Join("..", "..", "..", "testdata", "redaction_fixtures.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixtures not found at %s: %v", path, err)
	}

	var fixtures TestFixtures
	if err := json.Unmarshal(data, &fixtures); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			pos := int(syntaxErr.Offset) - 1
			if pos < 0 {
				pos = 0
			}
			if pos > len(data) {
				pos = len(data)
			}

			line := bytes.Count(data[:pos], []byte{'\n'}) + 1
			lastNL := bytes.LastIndex(data[:pos], []byte{'\n'})
			col := pos + 1
			if lastNL >= 0 {
				col = pos - lastNL
			}

			start := pos - 60
			if start < 0 {
				start = 0
			}
			end := pos + 60
			if end > len(data) {
				end = len(data)
			}
			snippet := strings.ReplaceAll(string(data[start:end]), "\n", "\\n")
			t.Fatalf("failed to parse fixtures: %v (line %d, col %d) near %q", err, line, col, snippet)
		}
		t.Fatalf("failed to parse fixtures: %v", err)
	}

	// Replace placeholders with runtime-generated synthetic values to avoid
	// GitHub secret scanning / push protection.
	replacer := strings.NewReplacer(
		"<<OPENAI_TEST_KEY>>", testOpenAIKey(),
		"<<OPENAI_LEGACY_KEY>>", testOpenAILegacyKey(),
		"<<OPENAI_PROJ_KEY>>", testOpenAIProjKey(),
		"<<ANTHROPIC_KEY>>", testAnthropicKey(),
		"<<GITHUB_TOKEN>>", testGitHubToken(),
		"<<GITHUB_FINE_PAT>>", testGitHubFinePAT(),
		"<<GITHUB_OAUTH_TOKEN>>", testGitHubOAuthToken(),
		"<<GITHUB_APP_TOKEN>>", testGitHubAppToken(),
		"<<AWS_ACCESS_KEY>>", testAWSAccessKey(),
		"<<AWS_STS_ACCESS_KEY>>", testAWSSTSAccessKey(),
		"<<AWS_SECRET_KEY>>", testAWSSecretKey(),
		"<<GOOGLE_API_KEY>>", testGoogleAPIKey(),
		"<<RSA_PRIVATE_KEY>>", testRSAPrivateKeyBlock(),
		"<<OPENSSH_PRIVATE_KEY>>", testOpenSSHPrivateKeyBlock(),
		"<<GENERIC_PRIVATE_KEY>>", testGenericPrivateKeyBlock(),
		"<<JWT>>", testJWT(),
	)

	for i := range fixtures.TruePositives {
		fixtures.TruePositives[i].Input = replacer.Replace(fixtures.TruePositives[i].Input)
	}
	for i := range fixtures.TrueNegatives {
		fixtures.TrueNegatives[i].Input = replacer.Replace(fixtures.TrueNegatives[i].Input)
	}
	for i := range fixtures.EdgeCases {
		fixtures.EdgeCases[i].Input = replacer.Replace(fixtures.EdgeCases[i].Input)
	}

	return &fixtures
}

func TestScanAndRedact_TruePositives(t *testing.T) {
	fixtures := loadFixtures(t)
	cfg := DefaultConfig()
	cfg.Mode = ModeWarn

	for _, tc := range fixtures.TruePositives {
		t.Run(tc.Description, func(t *testing.T) {
			result := ScanAndRedact(tc.Input, cfg)

			if len(result.Findings) == 0 {
				t.Errorf("expected to detect %s, got no findings", tc.ExpectedCategory)
				return
			}

			// Check that the expected category was found
			found := false
			for _, f := range result.Findings {
				if string(f.Category) == tc.ExpectedCategory {
					found = true
					break
				}
			}
			if !found {
				categories := make([]string, len(result.Findings))
				for i, f := range result.Findings {
					categories[i] = string(f.Category)
				}
				t.Errorf("expected category %s, got %v", tc.ExpectedCategory, categories)
			}
		})
	}
}

func TestScanAndRedact_TrueNegatives(t *testing.T) {
	fixtures := loadFixtures(t)
	cfg := DefaultConfig()
	cfg.Mode = ModeWarn

	for _, tc := range fixtures.TrueNegatives {
		t.Run(tc.Description, func(t *testing.T) {
			result := ScanAndRedact(tc.Input, cfg)

			if len(result.Findings) > 0 {
				categories := make([]string, len(result.Findings))
				for i, f := range result.Findings {
					categories[i] = string(f.Category)
				}
				t.Errorf("expected no findings, got %v", categories)
			}
		})
	}
}

func TestScanAndRedact_ModeOff(t *testing.T) {
	input := testOpenAIKey()
	cfg := Config{Mode: ModeOff}

	result := ScanAndRedact(input, cfg)

	if result.Output != input {
		t.Errorf("expected output unchanged, got %s", result.Output)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected no findings in off mode, got %d", len(result.Findings))
	}
}

func TestScanAndRedact_ModeWarn(t *testing.T) {
	input := "key: " + testOpenAIKey()
	cfg := Config{Mode: ModeWarn}

	result := ScanAndRedact(input, cfg)

	if result.Output != input {
		t.Errorf("expected output unchanged in warn mode")
	}
	if len(result.Findings) == 0 {
		t.Error("expected findings in warn mode")
	}
	if result.Blocked {
		t.Error("should not be blocked in warn mode")
	}
}

func TestScanAndRedact_ModeRedact(t *testing.T) {
	key := testOpenAIKey()
	input := "key: " + key
	cfg := Config{Mode: ModeRedact}

	result := ScanAndRedact(input, cfg)

	if strings.Contains(result.Output, key) {
		t.Error("expected key to be redacted")
	}
	if !strings.Contains(result.Output, "[REDACTED:OPENAI_KEY:") {
		t.Errorf("expected redaction placeholder, got %s", result.Output)
	}
	if len(result.Findings) == 0 {
		t.Error("expected findings")
	}
}

func TestScanAndRedact_ModeBlock(t *testing.T) {
	input := testOpenAIKey()
	cfg := Config{Mode: ModeBlock}

	result := ScanAndRedact(input, cfg)

	if !result.Blocked {
		t.Error("expected blocked=true")
	}
	if result.Output != input {
		t.Error("output should be unchanged in block mode")
	}
}

func TestScanAndRedact_Allowlist(t *testing.T) {
	// Test that allowlisted patterns are not flagged
	key := testOpenAIKey()
	input := key
	cfg := Config{
		Mode:      ModeWarn,
		Allowlist: []string{"^" + regexp.QuoteMeta(key) + "$"},
	}

	result := ScanAndRedact(input, cfg)

	// The key should be allowlisted and not reported
	if len(result.Findings) > 0 {
		t.Errorf("expected allowlisted key to not be flagged, got %d findings", len(result.Findings))
	}
}

func TestScanAndRedact_Allowlist_NoMatch(t *testing.T) {
	// Test that non-matching allowlist doesn't suppress findings
	input := testOpenAIKey()
	cfg := Config{
		Mode:      ModeWarn,
		Allowlist: []string{`^nope$`},
	}

	result := ScanAndRedact(input, cfg)

	// The key should still be detected
	if len(result.Findings) == 0 {
		t.Error("expected key to be detected when allowlist doesn't match")
	}
}

func TestScanAndRedact_DisabledCategories(t *testing.T) {
	input := testOpenAIKey() + " " + testAWSAccessKey()
	cfg := Config{
		Mode:               ModeWarn,
		DisabledCategories: []Category{CategoryOpenAIKey},
	}

	result := ScanAndRedact(input, cfg)

	for _, f := range result.Findings {
		if f.Category == CategoryOpenAIKey {
			t.Error("OpenAI key should be disabled")
		}
	}

	// AWS key should still be detected
	found := false
	for _, f := range result.Findings {
		if f.Category == CategoryAWSAccessKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("AWS key should be detected")
	}
}

func TestRedactionPlaceholder_Deterministic(t *testing.T) {
	input := "key: " + testOpenAIKey()
	cfg := Config{Mode: ModeRedact}

	r1 := ScanAndRedact(input, cfg)
	r2 := ScanAndRedact(input, cfg)

	if len(r1.Findings) == 0 || len(r2.Findings) == 0 {
		t.Fatalf("expected findings in both runs")
	}
	if r1.Findings[0].Redacted != r2.Findings[0].Redacted {
		t.Errorf("placeholders should be deterministic: %s != %s", r1.Findings[0].Redacted, r2.Findings[0].Redacted)
	}
}

func TestRedactionPlaceholder_Format(t *testing.T) {
	input := "key: " + testOpenAIKey()
	cfg := Config{Mode: ModeRedact}

	result := ScanAndRedact(input, cfg)
	if len(result.Findings) == 0 {
		t.Fatal("expected findings")
	}

	p := result.Findings[0].Redacted
	prefix := "[REDACTED:" + string(CategoryOpenAIKey) + ":"

	if !strings.HasPrefix(p, prefix) {
		t.Errorf("placeholder should start with %q, got %s", prefix, p)
	}
	if !strings.HasSuffix(p, "]") {
		t.Errorf("placeholder should end with ], got %s", p)
	}

	// Extract hash: remove prefix and suffix.
	hash := strings.TrimSuffix(strings.TrimPrefix(p, prefix), "]")
	if len(hash) != 8 {
		t.Errorf("hash should be 8 hex chars, got %d chars: %s", len(hash), hash)
	}

	// Verify hash is valid lowercase hex.
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hash should be lowercase hex, got: %s", hash)
			break
		}
	}
}

func TestAddLineInfo(t *testing.T) {
	input := "line1\nkey: " + testOpenAIKey() + "\nline3"
	cfg := Config{Mode: ModeWarn}

	result := ScanAndRedact(input, cfg)
	AddLineInfo(input, result.Findings)

	if len(result.Findings) == 0 {
		t.Fatal("expected findings")
	}

	f := result.Findings[0]
	if f.Line != 2 {
		t.Errorf("expected line 2, got %d", f.Line)
	}
	if f.Column < 5 {
		t.Errorf("expected column >= 5, got %d", f.Column)
	}
}

func TestContainsSensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal text", false},
		{testOpenAIKey(), true},
		{testAWSAccessKey(), true},
		{"", false},
	}

	cfg := DefaultConfig()
	for _, tc := range tests {
		result := ContainsSensitive(tc.input, cfg)
		if result != tc.expected {
			t.Errorf("ContainsSensitive(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestScan(t *testing.T) {
	input := "key: " + testOpenAIKey()
	cfg := DefaultConfig()

	findings := Scan(input, cfg)

	if len(findings) == 0 {
		t.Error("expected findings")
	}
}

func TestRedact(t *testing.T) {
	key := testOpenAIKey()
	input := "key: " + key
	cfg := DefaultConfig()

	output, findings := Redact(input, cfg)

	if strings.Contains(output, key) {
		t.Error("key should be redacted")
	}
	if len(findings) == 0 {
		t.Error("expected findings")
	}
}

func TestMultipleFindings(t *testing.T) {
	input := "OPENAI=" + testOpenAIKey() + " AWS=" + testAWSAccessKey()
	cfg := Config{Mode: ModeWarn}

	result := ScanAndRedact(input, cfg)

	if len(result.Findings) < 2 {
		t.Errorf("expected at least 2 findings, got %d", len(result.Findings))
	}

	categories := make(map[Category]bool)
	for _, f := range result.Findings {
		categories[f.Category] = true
	}

	if !categories[CategoryOpenAIKey] {
		t.Error("expected OPENAI_KEY finding")
	}
	if !categories[CategoryAWSAccessKey] {
		t.Error("expected AWS_ACCESS_KEY finding")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		mode    Mode
		wantErr bool
	}{
		{ModeOff, false},
		{ModeWarn, false},
		{ModeRedact, false},
		{ModeBlock, false},
		{"invalid", true},
	}

	for _, tc := range tests {
		cfg := Config{Mode: tc.mode}
		err := cfg.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("Validate(mode=%q) error = %v, wantErr %v", tc.mode, err, tc.wantErr)
		}
	}
}

func TestOverlappingMatches(t *testing.T) {
	// Test that overlapping patterns are handled correctly
	// Higher priority patterns should take precedence

	// This string could match both GENERIC_SECRET and OPENAI_KEY
	// OPENAI_KEY has higher priority and should win
	key := testOpenAIKey()
	input := "token=" + key
	cfg := Config{Mode: ModeRedact}

	result := ScanAndRedact(input, cfg)

	// Should only have one finding (OpenAI key, not generic secret)
	if len(result.Findings) != 1 {
		t.Errorf("expected exactly 1 finding (no overlaps), got %d", len(result.Findings))
		for _, f := range result.Findings {
			t.Logf("  - %s at [%d:%d]", f.Category, f.Start, f.End)
		}
	}

	// The finding should be categorized as OPENAI_KEY (higher priority)
	if len(result.Findings) > 0 && result.Findings[0].Category != CategoryOpenAIKey {
		t.Errorf("expected OPENAI_KEY category, got %s", result.Findings[0].Category)
	}

	// Verify redaction was applied
	if strings.Contains(result.Output, key) {
		t.Error("expected key to be redacted in output")
	}
}

func TestDeduplicationPreservesOrder(t *testing.T) {
	// Multiple non-overlapping secrets should all be found and in order
	input := "first=" + testAWSAccessKey() + " second=" + testOpenAIKey()
	cfg := Config{Mode: ModeWarn}

	result := ScanAndRedact(input, cfg)

	if len(result.Findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(result.Findings))
		return
	}

	// First finding should be AWS (earlier in string)
	if result.Findings[0].Category != CategoryAWSAccessKey {
		t.Errorf("first finding should be AWS, got %s", result.Findings[0].Category)
	}

	// Second should be OpenAI
	if result.Findings[1].Category != CategoryOpenAIKey {
		t.Errorf("second finding should be OPENAI_KEY, got %s", result.Findings[1].Category)
	}

	// Positions should be in order
	if result.Findings[0].Start >= result.Findings[1].Start {
		t.Error("findings should be ordered by position")
	}
}

func BenchmarkScanAndRedact(b *testing.B) {
	input := strings.Repeat("some normal text with no secrets ", 100)
	cfg := DefaultConfig()
	cfg.Mode = ModeRedact

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanAndRedact(input, cfg)
	}
}

func BenchmarkScanAndRedact_WithSecrets(b *testing.B) {
	input := "key: " + testOpenAIKey() + " and more " +
		strings.Repeat("text ", 100)
	cfg := DefaultConfig()
	cfg.Mode = ModeRedact

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanAndRedact(input, cfg)
	}
}
