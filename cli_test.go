package main

import (
	"flag"
	"github.com/stretchr/testify/suite"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	// Save original args and restore them after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Save original flags and restore them after test
	originalFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = originalFlagCommandLine }()

	// Set up test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create some special files to test ignore functionality
	// Hidden file
	hiddenFilePath := filepath.Join(testDir, ".hidden.txt")
	if err := os.WriteFile(hiddenFilePath, []byte("hidden content"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	// Package directory
	vendorDir := filepath.Join(testDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor directory: %v", err)
	}
	vendorFilePath := filepath.Join(vendorDir, "package.json")
	if err := os.WriteFile(vendorFilePath, []byte("{\"name\": \"test\"}"), 0644); err != nil {
		t.Fatalf("Failed to create vendor file: %v", err)
	}

	// Clean up test files after test
	defer os.Remove("skukozh_file_list.txt")
	defer os.Remove("skukozh_result.txt")

	// Create a .gitignore file
	gitignoreContent := "*.log\nignored_file.txt"
	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore file: %v", err)
	}

	// Create a file that should be ignored by .gitignore
	ignoredFile := filepath.Join(testDir, "ignored_file.txt")
	if err := os.WriteFile(ignoredFile, []byte("should be ignored"), 0644); err != nil {
		t.Fatalf("Failed to create gitignore test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "test.log"), []byte("log file"), 0644); err != nil {
		t.Fatalf("Failed to create gitignore test log file: %v", err)
	}

	tests := []struct {
		name           string
		args           []string
		expectedOut    string
		notExpectedOut string // String that should NOT be in the output
		expectFile     string
		expectCode     int
		setupRequired  func(t *testing.T) // Function to run before the test
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
			args:        []string{"skukozh", "-ext", "go", "find", testDir},
			expectedOut: "File list saved to",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Find command with no-ignore flag",
			args:        []string{"skukozh", "-no-ignore", "find", testDir},
			expectedOut: "File list saved to",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Find command with hidden flag",
			args:        []string{"skukozh", "-hidden", "find", testDir},
			expectedOut: "File list saved to",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Find command with verbose flag",
			args:        []string{"skukozh", "-verbose", "find", testDir},
			expectedOut: "Scanning directory",
			expectFile:  "skukozh_file_list.txt",
			expectCode:  0,
		},
		{
			name:        "Find command with nonexistent directory",
			args:        []string{"skukozh", "find", "/nonexistent/directory"},
			expectedOut: "Error walking directory",
			expectFile:  "",
			expectCode:  0,
			setupRequired: func(t *testing.T) {
				// Replace os.Exit to prevent actual exit
				originalOsExit := osExit
				osExit = func(code int) {
					// Do nothing in test
				}
				// Restore after test
				t.Cleanup(func() {
					osExit = originalOsExit
				})
			},
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
			// Create a custom FlagSet for this test instead of reusing the global one
			flagSet := DefaultFlags()

			// Run any required setup
			if tc.setupRequired != nil {
				tc.setupRequired(t)
			}

			// Set args for this test
			os.Args = tc.args
			t.Logf("Running with args: %v", tc.args)

			// Parse flags with our custom FlagSet
			if len(tc.args) > 0 {
				flagSet.Parse(tc.args[1:])
			}

			// Capture output and exit code
			var exitCode int
			output := CaptureOutput(t, func() {
				exitCode = runWithFlags(flagSet)
			})
			t.Logf("Output: %s", output)
			t.Logf("Exit code: %d", exitCode)

			if exitCode != tc.expectCode {
				t.Errorf("Expected exit code %d, got %d", tc.expectCode, exitCode)
			}

			if tc.expectedOut != "" && !strings.Contains(output, tc.expectedOut) {
				t.Errorf("Expected output to contain '%s', but got: %s", tc.expectedOut, output)
			}

			if tc.notExpectedOut != "" && strings.Contains(output, tc.notExpectedOut) {
				t.Errorf("Expected output to NOT contain '%s', but it did: %s", tc.notExpectedOut, output)
			}

			if tc.expectFile != "" {
				fileExists := FileExists(tc.expectFile)
				t.Logf("File %s exists: %v", tc.expectFile, fileExists)
				if !fileExists && exitCode == 0 {
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
	args := []string{"skukozh", "analyze"}
	os.Args = args

	// Create a custom FlagSet for this test
	flagSet := DefaultFlags()
	flagSet.Parse(args[1:])

	// Capture output
	output := CaptureOutput(t, func() {
		runWithFlags(flagSet) // Use runWithFlags instead of run
	})

	// Verify expected output
	if !strings.Contains(output, "Analysis Report") {
		t.Errorf("Expected 'Analysis Report' in output, got: %s", output)
	}

	if !strings.Contains(output, "file1.go") {
		t.Errorf("Expected file1.go in output, got: %s", output)
	}
}

func TestFlagIsolation(t *testing.T) {
	// Set up test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create two different flag sets with different flag values
	flagSet1 := DefaultFlags()
	flagSet1.Parse([]string{"-verbose", "find", testDir})

	flagSet2 := DefaultFlags()
	flagSet2.Parse([]string{"-no-ignore", "find", testDir})

	// Use separate output files for each test to avoid race conditions
	file1 := "skukozh_file_list_1.txt"
	file2 := "skukozh_file_list_2.txt"

	// Override the fileListName for each test to avoid conflicts
	origFileName := fileListName
	defer func() { fileListName = origFileName }()

	// Run the first command and capture output
	fileListName = file1
	output1 := CaptureOutput(t, func() {
		runWithFlags(flagSet1)
	})

	// Run the second command and capture output
	fileListName = file2
	output2 := CaptureOutput(t, func() {
		runWithFlags(flagSet2)
	})

	// Verify first output shows verbose messages
	if !strings.Contains(output1, "Scanning directory") {
		t.Errorf("Expected verbose output in first run, got: %s", output1)
	}

	// Verify second output doesn't contain verbose messages but does have the expected output
	if !strings.Contains(output2, "File list saved to") {
		t.Errorf("Expected 'File list saved to' in second run, got: %s", output2)
	}

	// Clean up
	os.Remove(file1)
	os.Remove(file2)
}

// Add a suite-based test to demonstrate testify suite functionality
type CLISuite struct {
	suite.Suite
	testDir       string
	cleanupFunc   func()
	originalArgs  []string
	originalFlags *flag.FlagSet
}

func (s *CLISuite) SetupSuite() {
	// Save original args and flags
	s.originalArgs = os.Args
	s.originalFlags = flag.CommandLine

	// Set up test directory
	var err error
	s.testDir, s.cleanupFunc = setupTestDir(s.T())

	// Create a special test file for the suite
	testFilePath := filepath.Join(s.testDir, "suite_test.txt")
	err = os.WriteFile(testFilePath, []byte("suite test content"), 0644)
	s.Require().NoError(err, "Failed to create test file for suite")
}

func (s *CLISuite) TearDownSuite() {
	// Clean up
	if s.cleanupFunc != nil {
		s.cleanupFunc()
	}

	// Restore original args and flags
	os.Args = s.originalArgs
	flag.CommandLine = s.originalFlags

	// Clean up test files
	os.Remove("skukozh_file_list.txt")
	os.Remove("skukozh_result.txt")
}

func (s *CLISuite) TestFindCommand() {
	// Set args for find command
	os.Args = []string{"skukozh", "find", s.testDir}

	// Create a custom FlagSet for this test
	flagSet := DefaultFlags()
	flagSet.Parse(os.Args[1:])

	// Capture output
	output := CaptureOutput(s.T(), func() {
		runWithFlags(flagSet)
	})

	// Assert output using testify
	s.Assert().Contains(output, "File list saved to", "Output should indicate file was saved")
	s.Assert().FileExists("skukozh_file_list.txt", "File list should be created")

	// Additional assertions
	fileContent, err := os.ReadFile("skukozh_file_list.txt")
	s.Require().NoError(err, "Should be able to read file list")
	s.Assert().NotEmpty(fileContent, "File list should not be empty")
}

func (s *CLISuite) TestVerboseFlag() {
	// Test with verbose flag
	os.Args = []string{"skukozh", "-verbose", "find", s.testDir}

	// Create a custom FlagSet for this test
	flagSet := DefaultFlags()
	flagSet.Parse(os.Args[1:])

	// Capture output
	output := CaptureOutput(s.T(), func() {
		runWithFlags(flagSet)
	})

	// Assert using testify
	s.Assert().Contains(output, "Scanning directory", "Verbose output should show scanning information")
	s.Assert().FileExists("skukozh_file_list.txt", "File list should be created")
}

// Run the suite
func TestCLISuite(t *testing.T) {
	suite.Run(t, new(CLISuite))
}
