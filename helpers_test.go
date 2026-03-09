package main

import (
	"slices"
	"testing"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
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

func TestValidateCommandMessage(t *testing.T) {
	// mock session's self.ID will be empty, so no userID is included here
	const mockShortPre = "!"
	const mockMention = "<@TESTBOT>"
	const mockMentionPrefix = mockMention + " " + mockShortPre
	mockSession := sgo.New("TESTBOT")
	const expectedCommand = "new"

	tests := []struct {
		name           string
		expectValid    bool
		input          *Context
		expectedPrefix string
	}{
		{
			name:        "lack of mention in public channel invalidated",
			expectValid: false,
			input: &Context{
				Message: &sgo.Message{
					Content: mockShortPre + "new 2026-12-25 $25",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
		},
		{
			name:        "lack of mention in dm channel not invalidated",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockShortPre + "new 2026-12-25 $25",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeDM,
				},
				Session: mockSession,
			},
			expectedPrefix: mockShortPre,
		},
		{
			name:        "missing command character invalidates",
			expectValid: false,
			input: &Context{
				Message: &sgo.Message{
					Content: mockMention + " new 2026-12-25 $25",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
		},
		{
			// mention, command character, and arguments present
			name:        "good command form validates in public channel",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockMentionPrefix + "new 2026-12-25 $25",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
			expectedPrefix: mockMentionPrefix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, command, _, isValid := validateCommandMessage(tt.input, (append(getValidPrefixes(tt.input), mockMentionPrefix)))
			if isValid != tt.expectValid {
				t.Errorf("ERROR: Expected validation status: %t, but got: %t", tt.expectValid, isValid)
			}
			if tt.expectValid {
				if tt.expectedPrefix != prefix {
					t.Errorf("ERROR: Expected prefix: %s, but got: %s", tt.expectedPrefix, prefix)
				}
				if expectedCommand != command {
					t.Errorf("ERROR: Expected command: %s, but got: %s", expectedCommand, command)
				}
			}
		})
	}
}
