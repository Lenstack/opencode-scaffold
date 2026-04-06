package core

import (
	"testing"
)

func TestStorageInterface(t *testing.T) {
	// Test that ErrNotFound is properly defined
	if ErrNotFound == nil {
		t.Error("expected ErrNotFound to be defined")
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{}
	if err.Error() != "not found" {
		t.Errorf("expected error message 'not found', got %q", err.Error())
	}
}

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{}
	msg := err.Error()
	if msg != "not found" {
		t.Errorf("expected 'not found', got %q", msg)
	}
}

func TestNotFoundError_Is(t *testing.T) {
	err1 := &NotFoundError{}
	err2 := &NotFoundError{}
	if err1 != err2 {
		// Different pointer instances - this is expected
	}
	// Test that the error can be compared
	if err1.Error() != err2.Error() {
		t.Errorf("expected same error message")
	}
}
