package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
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
