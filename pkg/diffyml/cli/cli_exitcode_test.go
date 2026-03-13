package cli

import (
	"errors"
	"fmt"
	"testing"
)

// Tests for exit code handling (Task 5.3)

func TestExitCodes_Constants(t *testing.T) {
	// Verify exit code constants match spec
	if ExitCodeSuccess != 0 {
		t.Errorf("expected ExitCodeSuccess=0, got %d", ExitCodeSuccess)
	}
	if ExitCodeDifferences != 1 {
		t.Errorf("expected ExitCodeDifferences=1, got %d", ExitCodeDifferences)
	}
	if ExitCodeError != 255 {
		t.Errorf("expected ExitCodeError=255, got %d", ExitCodeError)
	}
}

func TestDetermineExitCode_WithSetExitCode_NoDifferences(t *testing.T) {
	code := DetermineExitCode(true, 0, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d with -s and no differences, got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_HasDifferences(t *testing.T) {
	code := DetermineExitCode(true, 5, nil)
	if code != ExitCodeDifferences {
		t.Errorf("expected exit code %d with -s and differences, got %d", ExitCodeDifferences, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_HasError(t *testing.T) {
	code := DetermineExitCode(true, 0, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d with -s and error, got %d", ExitCodeError, code)
	}
}

func TestDetermineExitCode_WithSetExitCode_ErrorTakesPrecedence(t *testing.T) {
	// Error should take precedence over differences
	code := DetermineExitCode(true, 5, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d when error present, got %d", ExitCodeError, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_NoDifferences(t *testing.T) {
	code := DetermineExitCode(false, 0, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d without -s and no differences, got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_HasDifferences(t *testing.T) {
	// Without -s flag, should still return 0 even with differences
	code := DetermineExitCode(false, 5, nil)
	if code != ExitCodeSuccess {
		t.Errorf("expected exit code %d without -s flag (regardless of differences), got %d", ExitCodeSuccess, code)
	}
}

func TestDetermineExitCode_WithoutSetExitCode_HasError(t *testing.T) {
	// Error still returns error code even without -s flag
	code := DetermineExitCode(false, 0, fmt.Errorf("some error"))
	if code != ExitCodeError {
		t.Errorf("expected exit code %d on error, got %d", ExitCodeError, code)
	}
}

func TestExitResult_Success(t *testing.T) {
	result := NewExitResult(0, nil)
	if result.Code != ExitCodeSuccess {
		t.Errorf("expected code %d, got %d", ExitCodeSuccess, result.Code)
	}
	if result.Err != nil {
		t.Errorf("expected nil error, got %v", result.Err)
	}
	if !result.IsSuccess() {
		t.Error("expected IsSuccess() to return true")
	}
}

func TestExitResult_WithError(t *testing.T) {
	err := fmt.Errorf("test error")
	result := NewExitResult(ExitCodeError, err)
	if result.Code != ExitCodeError {
		t.Errorf("expected code %d, got %d", ExitCodeError, result.Code)
	}
	if !errors.Is(result.Err, err) {
		t.Errorf("expected error %v, got %v", err, result.Err)
	}
	if result.IsSuccess() {
		t.Error("expected IsSuccess() to return false")
	}
}

func TestExitResult_HasDifferences(t *testing.T) {
	result := NewExitResult(ExitCodeDifferences, nil)
	if result.Code != ExitCodeDifferences {
		t.Errorf("expected code %d, got %d", ExitCodeDifferences, result.Code)
	}
	if result.HasDifferences() != true {
		t.Error("expected HasDifferences() to return true")
	}
}

func TestExitResult_String(t *testing.T) {
	tests := []struct {
		code     int
		err      error
		contains string
	}{
		{ExitCodeSuccess, nil, "success"},
		{ExitCodeDifferences, nil, "differences"},
		{ExitCodeError, fmt.Errorf("parse failed"), "parse failed"},
	}

	for _, tc := range tests {
		result := NewExitResult(tc.code, tc.err)
		str := result.String()
		if !containsSubstr(str, tc.contains) {
			t.Errorf("expected String() to contain %q, got %q", tc.contains, str)
		}
	}
}
