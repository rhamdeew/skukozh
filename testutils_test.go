package main

import (
	"io"
	"os"
	"testing"
)

// CaptureOutput captures stdout during test execution
func CaptureOutput(t *testing.T, f func()) string {
	t.Helper()

	// Save the original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Replace stdout with the pipe writer
	os.Stdout = w

	// Run the function
	f()

	// Close the writer to signal we're done
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}

	// Restore original stdout
	os.Stdout = oldStdout

	// Read captured output
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	return string(out)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadTestFile reads a file for testing purposes
func ReadTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	return string(data)
}
