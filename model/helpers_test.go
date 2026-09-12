package model

import (
	"slices"
	"testing"
)

func TestShuffleStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
	}{
		{
			name:  "empty string remains empty",
			input: []string{},
		},
		{
			name:  "shuffled slice was shuffled",
			input: []string{"User1", "User2", "User3", "User4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputCopy := []string{}
			inputCopy = append(inputCopy, tt.input...)
			shuffleStrings(inputCopy)
			if len(tt.input) != len(inputCopy) {
				t.Errorf("length of shuffled slice (%d) != input slice (%d)", len(tt.input), len(inputCopy))
			}
			if len(tt.input) != 0 && slices.Equal(tt.input, inputCopy) {
				t.Errorf("slice was not shuffled")
			}
		})
	}
}
