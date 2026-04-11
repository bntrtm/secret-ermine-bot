package main

// 'sgo' as in "stoat go"
import (
	"fmt"
	"strings"

	sgo "github.com/sentinelb51/revoltgo"
)

// These constants may be used to points users to a public repository
// for the source code of this bot.
const (
	BotName           = "Secret Ermine"
	BotAvatarURL      = ""
	BotSourceCodeLink = "https://github.com/bntrtm/secret-ermine-bot"
)

// Context represents a single source of truth
// retaining all relevant details about a message event.
type Context struct {
	Session *sgo.Session
	Channel *sgo.Channel
	Server  *sgo.Server
	Caller  *sgo.User
	Message *sgo.Message
}

type Participant struct {
	SecretSanta string `json:"secret_santa"` // user tasked with getting this participant a gift
	Giftee      string `json:"giftee"`       // user this participant is tasked with giving a gift to
}

type SecretSantaEvent struct {
	Participants         map[string]Participant `json:"participants"`            // map of user IDs to participant info for all participating in the event
	OrganizationDate     string                 `json:"organization_date"`       // timestring referring to the date & time the session began
	DistributionDate     string                 `json:"distribution_date"`       // timestring referring to the date & time gifts will be distributed
	JoinMessageID        string                 `json:"join_message_id"`         // id of message participants are expected to react to
	JoinMessageChannelID string                 `json:"join_message_channel_id"` // channel where the join message can be found
	Notes                string                 `json:"notes"`                   // a user-input string providing further details surrounding the Secret Santa event
	Organizer            *sgo.User              `json:"organizer"`               // user that started the session
}

func (sse *SecretSantaEvent) hasStarted() bool {
	return len(sse.Participants) >= 3
}

// assignParticipants shuffles a slice of user IDs
// and then registers them as participants,
// assigning each a Giftee and Secret Santa.
func (sse *SecretSantaEvent) assignParticipants(uIDs []string) {
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

// printParticipantMap outputs the relationships of
// each participant to each other.
// EG, for each participant, outputs their giftee and
// Secret Santa.
// If a botStore pointer is provided, user IDs in the
// output will be converted to their human-readable
// usernames.
func (sse *SecretSantaEvent) printParticipantMapping(session *sgo.Session) {
	getName := func(uID string) string {
		if session != nil {
			user, err := getUser(session, uID)
			if err != nil {
				return uID
			}
			return user.Username
		} else {
			return uID
		}
	}

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

// details returns a multi-line string representing the
// details for this event
func (sse *SecretSantaEvent) details() string {
	return fmt.Sprintf("EVENT DETAILS:\n  - Distribution Date: %s\n  - Notes: %s", sse.DistributionDate, sse.Notes)
}

type ParticipantRelation int

const (
	Santa ParticipantRelation = iota
	Giftee
)

// Opp returns the corresponding value of the participant relation.
func (r *ParticipantRelation) Opp() ParticipantRelation {
	if *r == Santa {
		return Giftee
	}
	return Santa
}

// Title returns the formatted human-readable relation type
// for use in output or messages
func (r ParticipantRelation) Title() string {
	switch r {
	case Santa:
		return "Secret Santa"
	case Giftee:
		return "giftee"
	default:
		return "<UNKNOWN PARTICIPANT RELATION>"
	}
}

const ColourSoftRed = "#FF3939"
