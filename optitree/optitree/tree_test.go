package main

import (
	"slices"
	"testing"
)

func TestParseTreeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{name: "empty", input: "", want: []int{}},
		{name: "single", input: "1", want: []int{1}},
		{name: "multiple", input: "1,2,3", want: []int{1, 2, 3}},
		{name: "spaces", input: "1, 2, 3", want: []int{1, 2, 3}},
		{name: "mixed", input: "1, 2,3", want: []int{1, 2, 3}},
		{name: "mixed", input: "1, 2,3, 4", want: []int{1, 2, 3, 4}},
		{name: "with brackets no space", input: "[1,2,3,4]", want: []int{1, 2, 3, 4}},
		{name: "with brackets spaces", input: "[1, 2, 3, 4]", want: []int{1, 2, 3, 4}},
		{name: "with brackets", input: "[1, 2, 3, 4, 5]", want: []int{1, 2, 3, 4, 5}},
		{name: "with curly brackets no spaces", input: "{1,2,3,4,5}", want: []int{1, 2, 3, 4, 5}},
		{name: "with curly brackets spaces", input: "{1, 2, 3, 4, 5}", want: []int{1, 2, 3, 4, 5}},
		{name: "with []int curly brackets", input: "[]int{1, 2, 3, 4, 5, 6}", want: []int{1, 2, 3, 4, 5, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTreeString(tt.input)
			if err != nil {
				t.Errorf("parseTreeString(%q) returned error: %v", tt.input, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseTreeString(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewSubTree(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		root  int
		want  []int
	}{
		{name: "single", input: []int{1}, root: 1, want: []int{}},
		{name: "multiple", input: []int{1, 2, 3, 4}, root: 2, want: []int{1, 3, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inp := slices.Clone(tt.input)
			got := newSubtree(tt.root, tt.input)
			if !slices.Equal(tt.input, inp) {
				t.Errorf("newSubtree(%v, %d) modified input: %v", inp, tt.root, tt.input)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("newSubtree(%v, %d) = %v; want %v", tt.input, tt.root, got, tt.want)
			}
		})
	}
}
