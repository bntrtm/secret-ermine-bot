package main

import (
	"fmt"
	"strings"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

// botStore tracks persistent data related to the bot's activity across one or more servers
type botStore struct {
	Token               string
	Events              map[string]SecretSantaEvent    // map of servers to secret-santa events (limited to one active SSE/server)
	TrackedParticipants map[string]map[string]struct{} // map of user IDs to sets of Server IDs; useful in DM correspondence when bot needs to discern what the relevant event is
}

// findParticipantEvent takes a user ID and
// one or more characters that MAY correspond
// to a server ID and, if the user is found
// to be a participant in an event defined by
// a server ID that matches the prefix,
// returns the full server ID that the event is
// defined by.
// If the user is found to be the participant of
// only ONE event, and the prefix is empty, the
// one event will be returned.
func (b *botStore) findParticipantEvent(uID, sIDPrefix string) (string, error) {
	sIDs, ok := b.TrackedParticipants[uID]
	if !ok {
		return "", fmt.Errorf("user not found as participant in any event defined by sID prefix %s", sIDPrefix)
	}
	if len(sIDs) == 1 && sIDPrefix == "" {
		for sID := range sIDs {
			return sID, nil
		}
	}

	events := []string{}
	for sID := range sIDs {
		if strings.HasPrefix(sID, sIDPrefix) {
			events = append(events, sID)
		}
	}
	if len(events) == 1 {
		return events[0], nil
	} else {
		return "", fmt.Errorf("user found as participant in %d events defined by sID prefix %s", len(events), sIDPrefix)
	}
}

// syncEventParticipants syncs participants from an event
// defined by the given server ID with those whose IDs are
// used as keys in the TrackedParticipants map
func (b *botStore) syncEventParticipants(sID string) error {
	sse, ok := b.Events[sID]
	if !ok {
		return fmt.Errorf("trackEventParticipants(sID): no existing event defined by given server ID")
	}

	for k := range sse.Participants {
		// do we know this user is tracked at all?
		if _, ok := b.TrackedParticipants[k]; !ok {
			// if not, track them
			b.TrackedParticipants[k] = map[string]struct{}{}
		}
		// be sure we're tracking that they're a participant in this event
		b.TrackedParticipants[k][sID] = struct{}{}
	}

	return nil
}

// cleanTrackedParticipants updates tracked
// participants; if an event they're supposedly
// a part of doesn't exist, update their entry
// so that that's no longer the case.
// Clear the user as a participant entirely
// if they're then part of NO events.
func (b *botStore) cleanTrackedParticipants() {
	for uID, sIDs := range b.TrackedParticipants {
		for sID := range sIDs {
			if _, ok := b.Events[sID]; !ok {
				delete(b.TrackedParticipants[uID], sID)
			}
			if len(b.TrackedParticipants[uID]) == 0 {
				delete(b.TrackedParticipants, uID)
			}
		}
	}
}

func (b *botStore) sendDM(session *sgo.Session, sendMessage *sgo.MessageSend, userID string) error {
	dmChannel, err := session.DirectMessageCreate(userID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	message, err := session.ChannelMessageSend(dmChannel.ID, *sendMessage)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return err
	}

	fmt.Println("Sent message as DM to user: ", message.Content)
	return nil
}

// notifySantas messages all participants in the event known
// to the bot identified by the input server ID,
// letting them each know who their giftees are.
func (b *botStore) notifySantas(ctx *Context) error {
	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		return fmt.Errorf("could not find active event with given server ID")
	}

	embed := &sgo.MessageEmbed{}
	embed.Title = "🎅 Secret Santa Event"
	dsc := "**Welcome to %s's Secret Santa event from the %s server!**"
	dsc += "\nYour giftee is %s! They may send you a message soon **here** to give you an idea of what they might like as a gift."
	dsc += "\n\nYou should do the same for YOUR Secret Santa. To write a message to your Santa (be sure not to give yourself away!), you can do so in here by prefixing it with '!msg:santa'."
	dsc += "\n\n*Note that your giftee will not be the same person as your Santa.*\n?->Santa->You->Giftee->?"
	dsc += "\n" + sse.details()
	send := &sgo.MessageSend{
		Embeds: []*sgo.MessageEmbed{
			embed,
		},
	}

	sendCount := 0
	for uID, pt := range sse.Participants {
		giftee, err := getUser(ctx.Session, pt.Giftee)
		if err != nil {
			fmt.Println(err)
			continue
		}
		send.Embeds[0].Description = fmt.Sprintf(dsc, sse.Organizer.Mention(), ctx.Server.Name, giftee.Mention())
		err = b.sendDM(ctx.Session, send, uID)
		if err == nil {
			sendCount += 1
			fmt.Printf("Sent notifications to %d/%d Secret Santas.\n", sendCount, len(sse.Participants))
		}
	}

	return nil
}
