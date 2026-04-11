package main

import (
	"fmt"
	"sort"
	"strings"

	// 'sgo' as in "stoat go"
	"github.com/bntrtm/secret-ermine-bot/internal/logging"
	sgo "github.com/sentinelb51/revoltgo"
)

// botStore tracks persistent data related to the bot's activity across one or more servers
type botStore struct {
	logger *logging.Logger

	Events              map[string]SecretSantaEvent    // map of servers to secret-santa events (limited to one active SSE/server)
	TrackedParticipants map[string]map[string]struct{} // map of user IDs to sets of Server IDs; useful in DM correspondence when bot needs to discern what the relevant event is
	Token               string                         // bot token retrieved from environment variable
	AboutLinkParsed     bool                           // whether the value of BotSourceCodeLink properly parses as a URL
	Masquerade          *sgo.MessageMasquerade         // sent with all messages
	commands            map[string]command             // list of commands that may be run in one or both channel contexts
	commandKeys         []string                       // sorted list of command names
	platform            string                         // if DEV, dev-only commands are not exposed for use
}

// initCommands sets the bot's internal command list
// to their static values.
func (b *botStore) initCommands() {
	if len(b.commands) > 0 {
		return
	}
	b.commands = map[string]command{
		"help": {
			name:                  "help",
			description:           "Get help regarding bot usage.",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: true,
		},
		"new": {
			name:                  "new",
			description:           "Set up a new Secret Santa event in this server. Arguments: <Distribution Date *(YYYY-MM-DD)*> <Optional Notes... *(any text)*>",
			dmChannelsEnabled:     false,
			serverChannelsEnabled: true,
		},
		"start": {
			name:                  "start",
			description:           "Start a Secret Santa event, so long as three unique participants have offered a reaction to the join message.",
			dmChannelsEnabled:     false,
			serverChannelsEnabled: true,
		},
		"status": {
			name:                  "status",
			description:           "See the details of an existing Secret Santa event (or lack thereof) within this server.",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: true,
		},
		"cancel": {
			name:                  "cancel",
			description:           "Cancel an existing Secret Santa event in this server.",
			dmChannelsEnabled:     false,
			serverChannelsEnabled: true,
		},
		"dear-santa": {
			name:                  "dear-santa",
			description:           "Send a letter to your Secret Santa! Just follow it with the message you want to send!",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: false,
		},
		"dear-giftee": {
			name:                  "dear-giftee",
			description:           "Send a letter to your giftee! Just follow it with the message you want to send!",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: false,
		},
		"about": {
			name:                  "about",
			description:           "Get info about this bot instance",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: true,
		},
		"ping": {
			name:                  "ping",
			description:           "Check websocket latency with this bot.",
			dmChannelsEnabled:     true,
			serverChannelsEnabled: true,
			devOnly:               true,
		},
	}

	// get a sorted slice of command keys for later use
	commandKeys := make([]string, 0, len(b.commands))
	for k := range b.commands {
		commandKeys = append(commandKeys, k)
	}
	sort.Strings(commandKeys)
	b.commandKeys = commandKeys
}

// findParticipantEvent takes a user ID and
// one or more characters that MAY correspond
// to a server ID and returns a slice of events
// that the user is a participant of whose
// server IDs match the given prefix.
//
// An empty prefix returns all events that the
// participant is taking part in.
// one event will be returned.
func (b *botStore) findParticipantEvents(uID, sIDPrefix string) []string {
	sIDs, ok := b.TrackedParticipants[uID]
	if !ok {
		return []string{}
	}

	events := []string{}
	for sID := range sIDs {
		if strings.HasPrefix(sID, sIDPrefix) {
			events = append(events, sID)
		}
	}

	return events
}

// getParticipantEvent attempts to find an server ID matching
// an event that the user with the given user ID may be a part of,
// should the ID match the given prefix.
//
// The function returns the server ID, number of matching IDs,
// and any error upon return. The number of matches can be used to
// better determine and direct logic following one of either
// error case (no matches, multiple matches).
func (b *botStore) getParticipantEvent(uID, sIDPrefix string) (string, int, error) {
	events := b.findParticipantEvents(uID, sIDPrefix)

	switch len(events) {
	case 0:
		if sIDPrefix == "" {
			return "", 0, fmt.Errorf("user not found as participant in any events")
		}
		return "", 0, fmt.Errorf("user not found as participant in any event with sID prefix %s", sIDPrefix)
	case 1:
		return events[0], 1, nil
	default:
		return "", len(events), fmt.Errorf("user found as participant in %d events identified by sID prefix %s", len(events), sIDPrefix)
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
		}
		if len(b.TrackedParticipants[uID]) == 0 {
			delete(b.TrackedParticipants, uID)
		}
	}
}

func (b *botStore) sendDM(session *sgo.Session, sendMessage *sgo.MessageSend, userID string) error {
	if userID == "" {
		return fmt.Errorf("input user ID is empty")
	}

	dmChannel, err := session.DirectMessageCreate(userID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	sendMessage.Masquerade = b.Masquerade
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

	title := "🎅 Secret Santa Event"
	dsc := "**Welcome to %s's Secret Santa event from the %s server!**"
	dsc += "\nYour giftee is %s! They may send you a message soon **here** to give you an idea of what they might like as a gift."
	dsc += "\n\nYou should do the same for YOUR Secret Santa. To write a message to your Santa (be sure not to give yourself away!), you can do so in here by prefixing it with '!dear-santa'."
	dsc += "\nAs a Santa, you may also ask for clarifying information from your giftee by messaging them with the '!dear-giftee' command."
	dsc += "\n\n*Note that your giftee will not be the same person as your Santa.*\n?->Santa->You->Giftee->?"
	dsc += "\n" + sse.details()
	send := makeEmbeddedMessage(ColourSoftRed, title, dsc)

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

// TryServerLog wraps an event log function to validate context before
// attempting to log any message pertaining to a server-related event.
// This is useful for scopes wherein the context may pertain to DM
// channel rather than a server channel, in which case the logging
// ought be suppressed.
func (b *botStore) TryServerLog(logFunc func(sID, msg string, args ...any), ctx *Context, msg string, args ...any) {
	if ctx == nil {
		return
	}
	if ctx.Server == nil {
		return
	}
	logFunc(ctx.Server.ID, msg, args)
}

// Convenience wrapper calling TryServerLog with an ELogDebug func.
func (b *botStore) TryServerLogDebug(ctx *Context, msg string, args ...any) {
	b.TryServerLog(b.logger.ELogDebug, ctx, msg, args)
}

// Convenience wrapper calling TryServerLog with an ELogInfo func.
func (b *botStore) TryServerLogInfo(ctx *Context, msg string, args ...any) {
	b.TryServerLog(b.logger.ELogInfo, ctx, msg, args)
}

// Convenience wrapper calling TryServerLog with an ELogWarn func.
func (b *botStore) TryServerLogWarn(ctx *Context, msg string, args ...any) {
	b.TryServerLog(b.logger.ELogWarn, ctx, msg, args)
}

// Convenience wrapper calling TryServerLog with an ELogError func.
func (b *botStore) TryServerLogError(ctx *Context, msg string, args ...any) {
	b.TryServerLog(b.logger.ELogError, ctx, msg, args)
}
