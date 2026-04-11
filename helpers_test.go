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
	const mockServerPre = "!erm "
	mockSession := sgo.New("TESTBOT")
	const expectedCommand = "new"

	tests := []struct {
		name           string
		expectValid    bool
		input          *Context
		expectedPrefix string
	}{
		{
			name:        "use of shorthand prefix in public channel invalidates",
			expectValid: false,
			input: &Context{
				Message: &sgo.Message{
					Content: mockShortPre + "new 2026-12-25 $25 limit!",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
		},
		{
			name:        "use of shorthand prefix in dm channel does not invalidate",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockShortPre + "new 2026-12-25 $25 limit!",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeDM,
				},
				Session: mockSession,
			},
			expectedPrefix: mockShortPre,
		},
		{
			name:        "good command form in public channel validates",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockServerPre + "new 2026-12-25 $25 limit!",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
			expectedPrefix: mockServerPre,
		},
		{
			name:        "good command form in public ch validates despite extra space",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockServerPre + "new  2026-12-25 $25 limit!",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeText,
				},
				Session: mockSession,
			},
			expectedPrefix: mockServerPre,
		},
		{
			name:        "full command form in dm channel validates",
			expectValid: true,
			input: &Context{
				Message: &sgo.Message{
					Content: mockServerPre + "new 2026-12-25 $25 limit!",
				},
				Channel: &sgo.Channel{
					ChannelType: sgo.ChannelTypeDM,
				},
				Session: mockSession,
			},
			expectedPrefix: mockServerPre,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, command, _, isValid := validateCommandMessage(tt.input, (append(getValidPrefixes(tt.input), mockServerPre)))
			if isValid != tt.expectValid {
				t.Errorf("expected validation status: %t, but got: %t", tt.expectValid, isValid)
			}
			if tt.expectValid {
				if tt.expectedPrefix != prefix {
					t.Errorf("expected prefix: %s, but got: %s", tt.expectedPrefix, prefix)
				}
				if expectedCommand != command {
					t.Errorf("expected command: %s, but got: %s", expectedCommand, command)
				}
			}
		})
	}
}
