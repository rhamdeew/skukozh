package main

import (
	"flag"
	"os"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	// Save original args and restore them after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set up test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Clean up test files after test
	defer os.Remove("skukozh_file_list.txt")
	defer os.Remove("skukozh_result.txt")

	tests := []struct {
		name          string
		args          []string
		expectedOut   string
		expectFile    string
		expectCode    int
		setupRequired func(t *testing.T) // Function to run before the test
	}{
		{
			name:        "No arguments shows usage",
			args:        []string{"skukozh"},
			expectedOut: "Usage:",
			expectFile:  "",
			expectCode:  1,
		},
		{
			name:        "Find command",
			args:        []string{"skukozh", "find", testDir},
			expectedOut: "File list saved to",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Find command with extension filter",
			args:        []string{"skukozh", "-e", "go", "find", testDir},
			expectedOut: "File list saved to",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Generate command",
			args:        []string{"skukozh", "gen", testDir},
			expectedOut: "Content file saved",
			expectFile:  "skukozh_result.txt",
			expectCode:  0,
			setupRequired: func(t *testing.T) {
				// Create a file list for the generate command
				fileList := []string{
					"file1.go",
					"file2.js",
				}
				if err := os.WriteFile("skukozh_file_list.txt", []byte(strings.Join(fileList, "\n")), 0644); err != nil {
					t.Fatalf("Failed to create file list: %v", err)
				}
				t.Log("Created file list for generate command")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flags for each test
			flag.CommandLine.Init("skukozh", flag.ContinueOnError)

			// Run any required setup
			if tc.setupRequired != nil {
				tc.setupRequired(t)
			}

			// Set args for this test
			os.Args = tc.args
			t.Logf("Running with args: %v", tc.args)

			// Capture output and exit code
			var exitCode int
			output := CaptureOutput(t, func() {
				exitCode = run()
			})
			t.Logf("Output: %s", output)
			t.Logf("Exit code: %d", exitCode)

			if exitCode != tc.expectCode {
				t.Errorf("Expected exit code %d, got %d", tc.expectCode, exitCode)
			}

			if tc.expectedOut != "" && !strings.Contains(output, tc.expectedOut) {
				t.Errorf("Expected output to contain '%s', but got: %s", tc.expectedOut, output)
			}

			if tc.expectFile != "" {
				fileExists := FileExists(tc.expectFile)
				t.Logf("File %s exists: %v", tc.expectFile, fileExists)
				if !fileExists {
					t.Errorf("Expected file %s to be created but it wasn't", tc.expectFile)
				}
			}
		})

		// Clean up files between tests
		os.Remove("skukozh_file_list.txt")
		os.Remove("skukozh_result.txt")
	}
}

func TestAnalyzeCommand(t *testing.T) {
	// This test requires the result file to exist
	testContent := `#FILE file1.go
#TYPE go
#START
` + "```go" + `
package main
func main() {
  // This is a test
}
` + "```" + `
#END
`
	if err := os.WriteFile("skukozh_result.txt", []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test result file: %v", err)
	}
	defer os.Remove("skukozh_result.txt")

	// Save original args and restore them after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set args for analyze command
	os.Args = []string{"skukozh", "analyze"}

	// Reset flags
	flag.CommandLine.Init("skukozh", flag.ContinueOnError)

	// Capture output
	output := CaptureOutput(t, func() {
		run() // Call run() instead of main()
	})

	// Verify expected output
	if !strings.Contains(output, "Analysis Report") {
		t.Errorf("Expected 'Analysis Report' in output, got: %s", output)
	}

	if !strings.Contains(output, "file1.go") {
		t.Errorf("Expected file1.go in output, got: %s", output)
	}
}
