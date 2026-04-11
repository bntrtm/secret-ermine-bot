package main

import (
	"fmt"
	"strings"
	"testing"
	// 'sgo' as in "stoat go"
)

func TestGetParticipantEvent(t *testing.T) {
	const mockServerPrefix = "TEST"
	const mockServerID1 = mockServerPrefix + "-EVENT1"
	const mockServerID2 = mockServerPrefix + "-EVENT2"
	const outlierServerID = "OUTLIER-EVENT"
	users := []string{
		"User1",
		"User2",
		"User3",
		"User4",
	}

	bot := &botStore{
		Events: map[string]SecretSantaEvent{
			mockServerID1: {
				Participants: map[string]Participant{},
			},
			mockServerID2: {
				Participants: map[string]Participant{},
			},
			outlierServerID: {
				Participants: map[string]Participant{},
			},
		},
		TrackedParticipants: map[string]map[string]struct{}{},
	}
	for i, sID := range []string{mockServerID1, mockServerID2, outlierServerID} {
		sse, ok := bot.Events[sID]
		if !ok {
			t.Errorf("could not get event")
		}
		// event from mockServerID2 will not include User1
		// event from outlierServerID will not include User1 nor User2
		sse.assignParticipants(users[i:])
		bot.Events[sID] = sse

		err := bot.syncEventParticipants(sID)
		if err != nil {
			t.Errorf("could not sync event participants")
		}
	}

	t.Run("user not found", func(t *testing.T) {
		uID := "User0"
		_, _, err := bot.getParticipantEvent(uID, "")
		if err == nil {
			t.Errorf("expected user not found with empty prefix case, but apparently exists")
		}
		_, _, err = bot.getParticipantEvent(uID, mockServerPrefix)
		if err == nil {
			t.Errorf("expected user not found with case of prefix provided, but %s exists", uID)
		}
	})

	t.Run("empty prefix gets all events", func(t *testing.T) {
		// User4 with an empty prefix ought to yield three IDs
		_, matches, err := bot.getParticipantEvent(users[3], "")
		if err == nil || matches != 3 {
			t.Errorf("expected an error with three matches, but got: err: %t, matches: %d", err == nil, matches)
		}
	})

	// User4 with the "TEST-" prefix ought to yield two IDs
	t.Run("provided prefix w/ >1 results fails", func(t *testing.T) {
		sID, matches, err := bot.getParticipantEvent(users[3], mockServerPrefix)
		if err == nil {
			t.Errorf("expected error, got none")
		}
		if matches != 2 {
			t.Errorf("expected %d results, got %d", 2, matches)
		}
		if !strings.Contains(sID, ",") {
			t.Error("expected server ID string returned to contain two comma-separated IDs")
		}
	})

	t.Run("provided prefix w/ 1 result gets server ID", func(t *testing.T) {
		sID, matches, err := bot.getParticipantEvent(users[0], mockServerPrefix)
		if err != nil {
			t.Errorf("expected no error, got: %s", err)
		}
		if matches != 1 {
			t.Errorf("expected %d results, got %d", 1, matches)
		}
		if sID == "" {
			t.Error("expected single server ID in server ID string, got empty")
		} else if strings.Contains(sID, ",") {
			t.Error("expected single server ID in server ID string, got two or more comma-separated IDs")
		}
	})
}

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
