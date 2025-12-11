package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Helper to capture stdout/stderr
func captureOutput(f func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	f()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut bytes.Buffer
	var bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	return bufOut.String(), bufErr.String()
}

func TestFormatterErrors(t *testing.T) {
	var buf bytes.Buffer
	f := New(WithWriter(&buf), WithJSON(true))

	// Error
	err := errors.New("test error")
	f.Error(err)
	var resp ErrorResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON invalid: %v", err)
	}
	if resp.Error != "test error" {
		t.Errorf("Error() = %q, want 'test error'", resp.Error)
	}

	// ErrorMsg
	buf.Reset()
	f.ErrorMsg("msg error")
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON invalid: %v", err)
	}
	if resp.Error != "msg error" {
		t.Errorf("ErrorMsg() = %q, want 'msg error'", resp.Error)
	}

	// ErrorWithCode
	buf.Reset()
	f.ErrorWithCode("CODE", "msg")
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON invalid: %v", err)
	}
	if resp.Code != "CODE" {
		t.Errorf("ErrorWithCode() code = %q, want CODE", resp.Code)
	}
}

func TestPrintError(t *testing.T) {
	// Text mode
	_, stderr := captureOutput(func() {
		PrintError(errors.New("oops"), false)
	})
	if stderr != "Error: oops\n" {
		t.Errorf("PrintError text mode = %q, want 'Error: oops\n'", stderr)
	}

	// JSON mode
	stdout, _ := captureOutput(func() {
		PrintError(errors.New("json error"), true)
	})

	var resp ErrorResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Errorf("PrintError JSON invalid: %v", err)
	}
	if resp.Error != "json error" {
		t.Errorf("PrintError JSON error = %q, want 'json error'", resp.Error)
	}
}

func TestTimestamped(t *testing.T) {
	ts := NewTimestamped()
	if ts.GeneratedAt.IsZero() {
		t.Error("NewTimestamped() time is zero")
	}
	// Check if recent
	if time.Since(ts.GeneratedAt) > time.Second {
		t.Error("NewTimestamped() time is too old")
	}
}

func TestPrintJSON(t *testing.T) {
	data := map[string]string{"foo": "bar"}

	stdout, _ := captureOutput(func() {
		if err := PrintJSON(data); err != nil {
			t.Fatal(err)
		}
	})

	var resp map[string]string
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		t.Fatalf("PrintJSON output invalid: %v", err)
	}
	if resp["foo"] != "bar" {
		t.Errorf("PrintJSON foo = %q, want bar", resp["foo"])
	}
}

func TestOutputOrText(t *testing.T) {
	data := map[string]string{"key": "val"}
	textCalled := false
	textFn := func() error {
		textCalled = true
		return nil
	}

	// JSON mode
	// Note: OutputOrText writes to stdout for JSON
	captureOutput(func() {
		OutputOrText(true, data, textFn)
	})
	if textCalled {
		t.Error("OutputOrText(true) called textFn")
	}

	// Text mode
	OutputOrText(false, data, textFn)
	if !textCalled {
		t.Error("OutputOrText(false) did not call textFn")
	}
}

func TestFormatterMethods(t *testing.T) {
	f := New(WithPretty(true))

	// Writer
	if f.Writer() != os.Stdout {
		// Default is stdout
	}

	// Format
	if f.Format() != FormatText {
		t.Error("Expected FormatText default")
	}
}

func TestTextHelpersExtended(t *testing.T) {
	// Text
	var buf bytes.Buffer
	f := New(WithWriter(&buf))

	f.Text("hello")
	if buf.String() != "hello" {
		t.Errorf("Text() = %q, want hello", buf.String())
	}

	buf.Reset()
	f.Line()
	if buf.String() != "\n" {
		t.Errorf("Line() = %q, want newline", buf.String())
	}

	buf.Reset()
	f.Println("world")
	if buf.String() != "world\n" {
		t.Errorf("Println() = %q, want world\\n", buf.String())
	}

	buf.Reset()
	f.Printf("hello %s", "world")
	if buf.String() != "hello world" {
		t.Errorf("Printf() = %q, want hello world", buf.String())
	}
}

func TestTimestampFormat(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	s := FormatTime(now)
	if s != "2025-01-01T12:00:00Z" {
		t.Errorf("FormatTime() = %q", s)
	}

	parsed, err := ParseTime(s)
	if err != nil {
		t.Errorf("ParseTime() error: %v", err)
	}
	if !parsed.Equal(now) {
		t.Errorf("ParseTime() = %v, want %v", parsed, now)
	}
}

func TestDefaultFormatter(t *testing.T) {
	f := DefaultFormatter(true)
	if !f.IsJSON() {
		t.Error("DefaultFormatter(true) should be JSON")
	}

	f = DefaultFormatter(false)
	if f.IsJSON() {
		t.Error("DefaultFormatter(false) should be Text")
	}
}

func TestDetectFormatEnv(t *testing.T) {
	// Env var JSON
	os.Setenv("NTM_OUTPUT_FORMAT", "json")
	if f := DetectFormat(false); f != FormatJSON {
		t.Error("DetectFormat(false) with env=json should be JSON")
	}

	// Env var TEXT
	os.Setenv("NTM_OUTPUT_FORMAT", "text")
	if f := DetectFormat(false); f != FormatText {
		t.Error("DetectFormat(false) with env=text should be Text")
	}
	os.Unsetenv("NTM_OUTPUT_FORMAT")
}

func TestTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "Col1", "Col2")
	tbl.AddRow("Short", "Long Value Here")
	tbl.Render()

	output := buf.String()
	if !strings.Contains(output, "Col1") {
		t.Error("Table missing header Col1")
	}
	// Check for padding/alignment (heuristic)
	if !strings.Contains(output, "Short ") { // Should have padding
		t.Error("Table row padding seems missing")
	}
}
