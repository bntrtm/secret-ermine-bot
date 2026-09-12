package model

import (
	"fmt"
	"strings"
)

type SecretSantaEvent struct {
	Participants         map[string]Participant `json:"participants"`            // map of user IDs to participant info for all participating in the event
	OrganizationDate     string                 `json:"organization_date"`       // timestring referring to the date & time the session began
	DistributionDate     string                 `json:"distribution_date"`       // timestring referring to the date & time gifts will be distributed
	JoinMessageID        string                 `json:"join_message_id"`         // id of message participants are expected to react to
	JoinMessageChannelID string                 `json:"join_message_channel_id"` // channel where the join message can be found
	Notes                string                 `json:"notes"`                   // a user-input string providing further details surrounding the Secret Santa event
	OrganizerID          string                 `json:"organizer_id"`            // user that started the session
}

func (sse *SecretSantaEvent) HasStarted() bool {
	return len(sse.Participants) >= 3
}

// assignParticipants shuffles a slice of user IDs
// and then registers them as participants,
// assigning each a Giftee and Secret Santa.
func (sse *SecretSantaEvent) AssignParticipants(uIDs []string) {
	shuffleStrings(uIDs)

	for i, uID := range uIDs {
		// get user as participant (automatic zero value if not in map)
		pt := sse.Participants[uID]
		gifteeIndex := i + 1
		// if I'm last, my giftee is the participant with index 0
		if i == len(uIDs)-1 {
			gifteeIndex = 0
		}
		pt.Giftee = uIDs[gifteeIndex]
		// get giftee as participant (automatic zero value if not in map)
		ptGiftee := sse.Participants[pt.Giftee]
		// tell the system I'm their Secret Santa...
		ptGiftee.SecretSanta = uID
		sse.Participants[pt.Giftee] = ptGiftee
		// ...and tell the system I know who my giftee is
		sse.Participants[uID] = pt
	}
}

// PrintParticipantMap outputs the relationships of each participant to each other.
// EG, for each participant, outputs their giftee and Secret Santa.
// If a botStore pointer is provided, user IDs in the output will be converted to
// their human-readable usernames.
func (sse *SecretSantaEvent) PrintParticipantMapping(getName func(uID string) string) {
	var output strings.Builder
	output.WriteString("PARTICIPANT MAPPING:")
	for uID, p := range sse.Participants {
		participant := getName(uID)
		giftee := getName(p.Giftee)
		santa := getName(p.SecretSanta)
		fmt.Fprintf(&output, "\n%s is the Secret Santa of giftee: %s. Their Secret Santa is: %s", participant, giftee, santa)
	}

	fmt.Println(output.String())
}

// Details returns a multi-line string representing the details for this event
func (sse *SecretSantaEvent) Details() string {
	return fmt.Sprintf("EVENT DETAILS:\n  - Distribution Date: %s\n  - Notes: %s", sse.DistributionDate, sse.Notes)
}
