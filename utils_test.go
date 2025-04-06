package main

import (
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
			if result != tc.expected {
				t.Errorf("contains(%v, %s) = %v, want %v", tc.slice, tc.item, result, tc.expected)
			}
		})
	}
}
