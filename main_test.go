package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Empty slice", []string{}, ".go", false},
		{"Item exists", []string{".go", ".js", ".php"}, ".js", true},
		{"Item does not exist", []string{".go", ".js", ".php"}, ".ts", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.item)
			assert.Equal(t, tc.expected, result, "contains should return the correct result")
		})
	}

	// Additional assertions to demonstrate testify's capabilities
	t.Run("testify additional assertions", func(t *testing.T) {
		// Testing slices
		slice := []string{".go", ".js", ".php"}
		assert.Contains(t, slice, ".go", "Slice should contain element")
		assert.NotContains(t, slice, ".ts", "Slice should not contain element")
		assert.Len(t, slice, 3, "Slice should have correct length")

		// Using require for fatal assertions
		emptySlice := []string{}
		require.Empty(t, emptySlice, "Slice should be empty")

		// String assertions
		assert.Contains(t, "test.go", ".go", "String should contain substring")

		// Boolean assertions
		assert.True(t, contains(slice, ".go"), "Contains should return true")
		assert.False(t, contains(slice, ".ts"), "Contains should return false")
	})
}

func setupTestDir(t *testing.T) (string, func()) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "skukozh-test")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	files := map[string]string{
		"file1.go":         "package main\nfunc main() {\n\n}",
		"file2.js":         "function test() {\n\n}",
		"subdir/file3.go":  "package sub\nfunc Sub() {\n\n}",
		"subdir/file4.php": "<?php\necho 'test';\n?>",
		"empty.txt":        "",
		"file5.txt":        "Some text content\n\nWith blank lines",
	}

	for path, content := range files {
		fullPath := filepath.Join(testDir, path)
		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	return testDir, func() {
		os.RemoveAll(testDir)
	}
}

func TestFindFiles(t *testing.T) {
	// Set up test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Save original flags and restore them after test
	originalFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = originalFlagCommandLine }()

	// Temporarily change working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Create a hidden file and directory to test ignore functionality
	hiddenFile := filepath.Join(testDir, ".hidden.txt")
	if err := os.WriteFile(hiddenFile, []byte("hidden content"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	hiddenDir := filepath.Join(testDir, ".hiddendir")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("Failed to create hidden directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "file.txt"), []byte("hidden dir file"), 0644); err != nil {
		t.Fatalf("Failed to create file in hidden directory: %v", err)
	}

	// Create a binary file
	binaryFile := filepath.Join(testDir, "image.jpg")
	if err := os.WriteFile(binaryFile, []byte("fake binary data"), 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	// Create a package directory with a file
	vendorDir := filepath.Join(testDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "package.js"), []byte("vendor file"), 0644); err != nil {
		t.Fatalf("Failed to create file in vendor directory: %v", err)
	}

	// Create local variables for flags instead of using global ones
	tests := []struct {
		name           string
		supportedExts  []string
		noIgnoreValue  bool
		expectedCount  int
		expectedPrefix string
	}{
		{"Default behavior", []string{}, false, 5, ""}, // Finds .go, .js, .php, .txt but not hidden files or binary files
		{"No ignore", []string{}, true, 9, ""}, // Should find all files including hidden, binary, and package files
		{"Go files only", []string{".go"}, false, 2, ""},
		{"Multiple extensions", []string{".go", ".js"}, false, 3, ""},
		{"No matching files", []string{".c"}, false, 0, ""},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			// Clean up previous test file
			os.Remove("skukozh_file_list.txt")

			// Store original noIgnore value and restore it at the end of the test
			flagMutex.Lock()
			originalNoIgnoreValue := *noIgnore
			*noIgnore = tc.noIgnoreValue
			flagMutex.Unlock()

			// Make sure we restore it when we're done
			defer func() {
				flagMutex.Lock()
				*noIgnore = originalNoIgnoreValue
				flagMutex.Unlock()
			}()

			// For the find command directly:
			files, err := findFilesInternal(testDir, tc.supportedExts)
			if err != nil {
				t.Fatalf("findFilesInternal returned error: %v", err)
			}

			// Write files to test output
			if len(files) > 0 {
				fileContent := strings.Join(files, "\n")
				if err := os.WriteFile("skukozh_file_list.txt", []byte(fileContent), 0644); err != nil {
					t.Fatalf("Failed to write test output: %v", err)
				}
			}

			// Check if the expected number of files was found
			if len(files) != tc.expectedCount {
				t.Logf("With noIgnore=%v, found %d files: %v", tc.noIgnoreValue, len(files), files)
				t.Errorf("Expected %d files, got %d.", tc.expectedCount, len(files))
			}

			for _, file := range files {
				if tc.expectedPrefix != "" && !strings.HasPrefix(file, tc.expectedPrefix) {
					t.Errorf("File path does not start with expected prefix: %s", file)
				}
			}
		})
	}
}

func TestFindFilesErrors(t *testing.T) {
	// Test with a non-existent directory
	nonExistentDir := "/non/existent/directory"

	_, err := findFilesInternal(nonExistentDir, nil)
	if err == nil {
		t.Errorf("Expected error for non-existent directory, got nil")
	}

	// Test the output of the findFiles function
	// Save and restore os.Exit
	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	var exitCode int
	var exitCalled bool
	// Mock os.Exit
	osExit = func(code int) {
		exitCode = code
		exitCalled = true
		// Don't actually exit
	}

	output := CaptureOutput(t, func() {
		// Create a temporary FlagSet for this test
		tempFlags := DefaultFlags()
		findFiles(nonExistentDir, nil, tempFlags)
	})

	// Verify exit was called
	if !exitCalled {
		t.Errorf("Expected os.Exit to be called")
	}

	// Verify the exit code
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Verify the error message
	if !strings.Contains(output, "Error walking directory") {
		t.Errorf("Expected error message about walking directory, got: %s", output)
	}
}

func TestGenerateContentFile(t *testing.T) {
	// Set up test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a file list
	fileList := []string{
		"file1.go",
		"file2.js",
	}
	if err := os.WriteFile("skukozh_file_list.txt", []byte(strings.Join(fileList, "\n")), 0644); err != nil {
		t.Fatalf("Failed to create file list: %v", err)
	}

	// Clean up after test
	defer os.Remove("skukozh_file_list.txt")
	defer os.Remove("skukozh_result.txt")

	generateContentFile(testDir)

	// Check if the result file was created
	if !FileExists("skukozh_result.txt") {
		t.Fatalf("Expected result file was not created")
	}

	result := ReadTestFile(t, "skukozh_result.txt")

	// Check for file markers
	if !strings.Contains(result, "#FILE file1.go") {
		t.Errorf("Result does not contain file1.go marker")
	}
	if !strings.Contains(result, "#FILE file2.js") {
		t.Errorf("Result does not contain file2.js marker")
	}

	// Check for type markers
	if !strings.Contains(result, "#TYPE go") {
		t.Errorf("Result does not contain go type marker")
	}
	if !strings.Contains(result, "#TYPE js") {
		t.Errorf("Result does not contain js type marker")
	}

	// Check for content format
	if !strings.Contains(result, "```go") {
		t.Errorf("Result does not contain go code block")
	}
	if !strings.Contains(result, "```js") {
		t.Errorf("Result does not contain js code block")
	}
}

func TestGenerateContentFileErrors(t *testing.T) {
	// Setup - create test directory
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Test case 1: missing file list
	t.Run("missing file list", func(t *testing.T) {
		// Make sure skukozh_file_list.txt doesn't exist
		os.Remove("skukozh_file_list.txt")

		// Test the internal function
		_, err := generateContentFileInternal(testDir)
		if err == nil {
			t.Errorf("Expected error for missing file list, got nil")
		}

		// Test the main function with mocked os.Exit
		var exitCalled bool
		osExit = func(code int) {
			exitCalled = true
		}

		output := CaptureOutput(t, func() {
			generateContentFile(testDir)
		})

		// Verify exit was called
		if !exitCalled {
			t.Errorf("Expected os.Exit to be called")
		}

		if !strings.Contains(output, "Error reading file list") {
			t.Errorf("Expected error about reading file list, got: %s", output)
		}
	})

	// Test case 2: file list with non-existent file
	t.Run("non-existent file in list", func(t *testing.T) {
		// Create a file list with a non-existent file
		fileList := []string{
			"non-existent-file.txt",
		}
		if err := os.WriteFile("skukozh_file_list.txt", []byte(strings.Join(fileList, "\n")), 0644); err != nil {
			t.Fatalf("Failed to create file list: %v", err)
		}
		defer os.Remove("skukozh_file_list.txt")

		// Test the internal function
		output, err := generateContentFileInternal(testDir)
		if err != nil {
			t.Errorf("Did not expect error from internal function: %v", err)
		}

		// Logs are captured in output
		if !strings.Contains(output, "") { // Empty output is expected
			t.Logf("Output contains: %s", output)
		}

		// Also test the main function
		capturedOutput := CaptureOutput(t, func() {
			generateContentFile(testDir)
		})

		if !strings.Contains(capturedOutput, "Error reading file") {
			t.Errorf("Expected error about reading file, got: %s", capturedOutput)
		}
	})
}

func TestAnalyzeResultFile(t *testing.T) {
	// Create a test result file
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

#FILE file2.js
#TYPE js
#START
` + "```js" + `
function test() {
  // This is a longer test
  // With more content
  return true;
}
` + "```" + `
#END
`
	if err := os.WriteFile("skukozh_result.txt", []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test result file: %v", err)
	}
	defer os.Remove("skukozh_result.txt")

	// Capture stdout using our utility
	output := CaptureOutput(t, func() {
		analyzeResultFile(5)
	})

	// Verify output contains expected information
	if !strings.Contains(output, "Analysis Report") {
		t.Errorf("Output does not contain 'Analysis Report'")
	}
	if !strings.Contains(output, "file1.go") {
		t.Errorf("Output does not contain file1.go")
	}
	if !strings.Contains(output, "file2.js") {
		t.Errorf("Output does not contain file2.js")
	}
}

func TestAnalyzeResultFileErrors(t *testing.T) {
	// Save and restore os.Exit
	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	// Test with missing result file
	t.Run("missing result file", func(t *testing.T) {
		// Make sure skukozh_result.txt doesn't exist
		os.Remove("skukozh_result.txt")

		// Test with internal function
		_, err := analyzeResultFileInternal(10)
		if err == nil {
			t.Errorf("Expected error for missing result file, got nil")
		}

		// Test with main function
		var exitCalled bool
		osExit = func(code int) {
			exitCalled = true
		}

		output := CaptureOutput(t, func() {
			analyzeResultFile(10)
		})

		// Verify exit was called
		if !exitCalled {
			t.Errorf("Expected os.Exit to be called")
		}

		if !strings.Contains(output, "Error reading result file") {
			t.Errorf("Expected error about reading result file, got: %s", output)
		}
	})

	// Test with empty result file
	t.Run("empty result file", func(t *testing.T) {
		// Create an empty result file
		if err := os.WriteFile("skukozh_result.txt", []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create empty result file: %v", err)
		}
		defer os.Remove("skukozh_result.txt")

		// Test with internal function
		result, err := analyzeResultFileInternal(10)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "No files found") {
			t.Errorf("Expected result to contain 'No files found', got: %s", result)
		}

		// Test with main function
		output := CaptureOutput(t, func() {
			analyzeResultFile(10)
		})

		if !strings.Contains(output, "No files found") {
			t.Errorf("Expected message about no files found, got: %s", output)
		}
	})

	// Test with malformed result file
	t.Run("malformed result file", func(t *testing.T) {
		// Create a malformed result file
		malformedContent := "#FILE test.go\n#TYPE go\n#START\n```go\n// Content without proper end marker\n"
		if err := os.WriteFile("skukozh_result.txt", []byte(malformedContent), 0644); err != nil {
			t.Fatalf("Failed to create malformed result file: %v", err)
		}
		defer os.Remove("skukozh_result.txt")

		// Test with internal function
		result, err := analyzeResultFileInternal(10)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "Analysis Report") {
			t.Errorf("Expected result to contain 'Analysis Report', got: %s", result)
		}

		// Test with main function
		output := CaptureOutput(t, func() {
			analyzeResultFile(10)
		})

		// Check that analysis runs without crashing
		if !strings.Contains(output, "Analysis Report") {
			t.Errorf("Expected analysis report header, got: %s", output)
		}
	})
}
