package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Formatter.Error / ErrorMsg / ErrorWithCode — text mode branches (66.7% → 100%)
// ---------------------------------------------------------------------------

func TestFormatterError_TextMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := New(WithWriter(&buf), WithJSON(false))

	err := errors.New("text error")
	retErr := f.Error(err)
	if retErr == nil {
		t.Fatal("expected non-nil error return")
	}
	if retErr.Error() != "text error" {
		t.Errorf("expected original error, got %q", retErr.Error())
	}
	// In text mode, Error returns the error without writing to buffer.
	if buf.Len() > 0 {
		t.Errorf("expected no output in text mode, got %q", buf.String())
	}
}

func TestFormatterErrorMsg_TextMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := New(WithWriter(&buf), WithJSON(false))

	retErr := f.ErrorMsg("text msg error")
	if retErr == nil {
		t.Fatal("expected non-nil error return")
	}
	if !strings.Contains(retErr.Error(), "text msg error") {
		t.Errorf("expected error message, got %q", retErr.Error())
	}
}

func TestFormatterErrorWithCode_TextMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	f := New(WithWriter(&buf), WithJSON(false))

	retErr := f.ErrorWithCode("ERR_TEST", "coded error")
	if retErr == nil {
		t.Fatal("expected non-nil error return")
	}
	if !strings.Contains(retErr.Error(), "ERR_TEST") {
		t.Errorf("expected error code in message, got %q", retErr.Error())
	}
	if !strings.Contains(retErr.Error(), "coded error") {
		t.Errorf("expected error message, got %q", retErr.Error())
	}
}

// ---------------------------------------------------------------------------
// Steps nil-current guards (Done/Fail/Skip/Warn at 80% → 100%)
// ---------------------------------------------------------------------------

func TestSteps_Done_NilCurrent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	s := NewStepsWriter(&buf)
	// Call Done without calling Start — should not panic.
	s.Done()
	if buf.Len() > 0 {
		t.Errorf("expected no output for nil current, got %q", buf.String())
	}
}

func TestSteps_Fail_NilCurrent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	s := NewStepsWriter(&buf)
	s.Fail()
	if buf.Len() > 0 {
		t.Errorf("expected no output for nil current, got %q", buf.String())
	}
}

func TestSteps_Skip_NilCurrent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	s := NewStepsWriter(&buf)
	s.Skip()
	if buf.Len() > 0 {
		t.Errorf("expected no output for nil current, got %q", buf.String())
	}
}

func TestSteps_Warn_NilCurrent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	s := NewStepsWriter(&buf)
	s.Warn()
	if buf.Len() > 0 {
		t.Errorf("expected no output for nil current, got %q", buf.String())
	}
}
