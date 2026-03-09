package main

import (
	"fmt"
	"testing"
	// 'sgo' as in "stoat go"
)

func TestSyncTrackedParticipants(t *testing.T) {
	const mockServerID = "TEST-EVENT"
	users := []string{
		"User1",
		"User2",
		"User3",
		"User4",
	}

	bot := &botStore{
		Events: map[string]SecretSantaEvent{
			mockServerID: {
				Participants: map[string]Participant{},
				SpendLimit:   "$25",
			},
		},
		TrackedParticipants: map[string]map[string]struct{}{},
	}
	sse, ok := bot.Events[mockServerID]
	if !ok {
		t.Errorf("could not get event")
	}
	sse.assignParticipants(users)
	bot.Events[mockServerID] = sse

	err := bot.syncEventParticipants(mockServerID)
	if err != nil {
		t.Errorf("could not sync event participants")
	}

	for _, uID := range users {
		if _, ok := bot.TrackedParticipants[uID]; !ok {
			t.Errorf("participant expected, but not present: %s", uID)
		}
	}
}

func TestCleanTrackedParticipants(t *testing.T) {
	const mockServerID = "TEST-EVENT"
	users := []string{
		"User1",
		"User2",
		"User3",
		"User4",
	}

	bot := &botStore{
		Events: map[string]SecretSantaEvent{
			mockServerID: {
				Participants: map[string]Participant{},
				SpendLimit:   "$25",
			},
		},
		TrackedParticipants: map[string]map[string]struct{}{
			// ensure users tracked from old events are cleared
			"User5": {
				"OLD-EVENT": {},
			},
			// ensure users with no events assigned are cleared
			"User6": {},
		},
	}
	sse, ok := bot.Events[mockServerID]
	if !ok {
		t.Errorf("could not get event")
	}
	sse.assignParticipants(users)
	bot.Events[mockServerID] = sse

	err := bot.syncEventParticipants(mockServerID)
	if err != nil {
		t.Errorf("could not sync event participants")
	}

	bot.cleanTrackedParticipants()

	for _, uID := range users {
		if _, ok := bot.TrackedParticipants[uID]; !ok {
			t.Errorf("participant expected, but must have been cleared: %s", uID)
		}
	}
	for _, uID := range []string{"User5", "User6"} {
		if _, ok := bot.TrackedParticipants[uID]; ok {
			fmt.Println(bot.TrackedParticipants[uID])
			t.Errorf("orphaned participant not cleared as expected: %s", uID)
		}
	}
}
