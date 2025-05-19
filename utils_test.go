package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestContainsFull(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Empty slice", []string{}, "item", false},
		{"Single item exists", []string{"item"}, "item", true},
		{"Single item doesn't exist", []string{"item"}, "other", false},
		{"Multiple items, exists at start", []string{"item", "other", "another"}, "item", true},
		{"Multiple items, exists in middle", []string{"other", "item", "another"}, "item", true},
		{"Multiple items, exists at end", []string{"other", "another", "item"}, "item", true},
		{"Multiple items, doesn't exist", []string{"other", "another", "something"}, "item", false},
		{"Case sensitive", []string{"Item"}, "item", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.item)
			assert.Equal(t, tc.expected, result, "contains(%v, %s) returned unexpected result", tc.slice, tc.item)
		})
	}
}

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"Hidden file", ".gitignore", true},
		{"Hidden directory", ".git", true},
		{"Non-hidden file", "main.go", false},
		{"Non-hidden directory", "src", false},
		{"Hidden file with directory", ".config/file", true},
		{"Non-hidden with dot in name", "main.go.bak", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isHidden(tc.filename)
			assert.Equal(t, tc.expected, result, "isHidden(%s) returned unexpected result", tc.filename)
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Empty slice", []string{}, "item", false},
		{"Single item exists exact match", []string{"item"}, "item", true},
		{"Single item exists case insensitive", []string{"Item"}, "item", true},
		{"Single item exists mixed case", []string{"iTem"}, "iTeM", true},
		{"Multiple items, exists case insensitive", []string{"other", "Item", "another"}, "item", true},
		{"Multiple items, doesn't exist", []string{"other", "another", "something"}, "item", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := containsIgnoreCase(tc.slice, tc.item)
			assert.Equal(t, tc.expected, result, "containsIgnoreCase(%v, %s) returned unexpected result", tc.slice, tc.item)
		})
	}
}

func TestParseGitignore(t *testing.T) {
	// Create a temporary .gitignore file
	tempDir, err := os.MkdirTemp("", "gitignore-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	gitignoreContent := `
# This is a comment
*.log
node_modules/
/root_only.txt
!important.log
dir/subdir/*.txt
`
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create test .gitignore file: %v", err)
	}

	rules, err := parseGitignore(gitignorePath)
	assert.NoError(t, err)
	assert.Len(t, rules, 5, "Should have parsed 5 rules")

	// Check specific rules
	foundLogRule := false
	foundNodeModulesRule := false
	foundNegatedRule := false

	for _, rule := range rules {
		if rule.pattern == "*.log" && !rule.isDir && !rule.isNegated {
			foundLogRule = true
		}
		if rule.pattern == "node_modules" && rule.isDir && !rule.isNegated {
			foundNodeModulesRule = true
		}
		if rule.pattern == "important.log" && rule.isNegated {
			foundNegatedRule = true
		}
	}

	assert.True(t, foundLogRule, "Should have found *.log rule")
	assert.True(t, foundNodeModulesRule, "Should have found node_modules/ directory rule")
	assert.True(t, foundNegatedRule, "Should have found negated !important.log rule")
}

func TestMatchGitignorePattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"Exact match", "file.txt", "file.txt", true},
		{"Directory match", "dir/file.txt", "dir", true},
		{"Single wildcard", "file.log", "*.log", true},
		{"Single wildcard no match", "file.txt", "*.log", false},
		{"Double wildcard", "dir/subdir/file.txt", "dir/**/file.txt", true},
		{"Double wildcard with extension", "dir/subdir/file.log", "**/*.log", true},
		{"Double wildcard non-match", "dir/subdir/file.txt", "dir/**/other.txt", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchGitignorePattern(tc.path, tc.pattern)
			assert.Equal(t, tc.expected, result, "matchGitignorePattern(%s, %s) returned unexpected result", tc.path, tc.pattern)
		})
	}
}

func TestIsIgnoredByGitignore(t *testing.T) {
	rules := []gitignoreRule{
		{pattern: "*.log", isDir: false, isNegated: false},
		{pattern: "node_modules", isDir: true, isNegated: false},
		{pattern: "important.log", isDir: false, isNegated: true},
		{pattern: "build", isDir: true, isNegated: false},
	}

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"Log file should be ignored", "error.log", false, true},
		{"Important log should not be ignored", "important.log", false, false},
		{"Normal file should not be ignored", "file.txt", false, false},
		{"Node modules dir should be ignored", "node_modules", true, true},
		{"File in node_modules should be ignored", "node_modules/package.json", false, true},
		{"Build dir should be ignored", "build", true, true},
		{"File in build dir should be ignored", "build/output.js", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isIgnoredByGitignore(tc.path, rules, tc.isDir)
			assert.Equal(t, tc.expected, result, "isIgnoredByGitignore(%s, rules, %v) returned unexpected result", tc.path, tc.isDir)
		})
	}
}
