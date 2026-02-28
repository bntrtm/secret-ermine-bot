package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

func (b *botStore) handlerEventMessage(session *sgo.Session, msg *sgo.EventMessage) {
	var content string
	recordJoinMessage := false

	// always try to send ANY existing message in the content buffer, if present
	defer func() {
		if content == "" {
			return
		}
		send := sgo.MessageSend{
			Content: content,
		}

		message, err := b.session.ChannelMessageSend(msg.Channel, send)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}

		fmt.Println("Sent message: ", message.Content)

		if recordJoinMessage {
			server, err := b.getServerByChannelID(message.Channel)
			if err != nil {
				return
			}
			if entry, ok := b.Events[server.ID]; ok {
				entry.JoinMessageChannelID = message.Channel
				entry.JoinMessageID = message.ID
				b.Events[server.ID] = entry
			}
		}
	}()

	// command calls to the bot must first mention it
	self := b.session.State.Self()
	expectedPrefix := fmt.Sprintf("%s !", self.Mention())

	if !strings.HasPrefix(msg.Content, expectedPrefix) {
		return
	}

	fields := strings.Split(msg.Content, " ")
	command, args := strings.TrimPrefix(fields[1], "!"), fields[2:]
	switch command {
	case "new":
		success := false
		content, success = b.handleNewSantaEventMessage(args, msg)
		recordJoinMessage = success
	case "start":
		content = b.handleMsgStart(msg)
	case "status":
		content = b.handleMsgStatus(msg)
	case "help":
		content = b.handleMsgHelp()
	case "ping":
		content = b.handleMsgPing()
	case "cancel":
		content = b.handleMsgCancel(msg)
	default:
		content = fmt.Sprintf("Unknown command '%s', use '!help' for all available commands.", fields[0])
	}
}

// handlerNewSantaSession tells the bot it's time for a new Secret Santa Session!
// usage: !new <date (YYYY-MM-DD)> <spend_limit>
func (b *botStore) handleNewSantaEventMessage(args []string, msg *sgo.EventMessage) (string, bool) {
	server, err := b.getServerByChannelID(msg.Channel)
	if err != nil {
		return "", false
	}

	if event, ok := b.Events[server.ID]; ok {
		return fmt.Sprintf("A Secret Santa event organized by %s is already active in this server.\nThey must use the '!cancel' command before setting up a new one.", event.Organizer.Mention()), false
	}

	var content string

	if len(args) != 2 {
		content = fmt.Sprintf("Argument mismatch; expected 2, but got %d", len(args))
		return content, false
	}

	dateInput := args[0]
	spendLimit := args[1]

	distributionDate, err := time.Parse("2006-01-02", dateInput)
	if err != nil {
		fmt.Println("Could not parse distribution date provided as time.Time")
		content = fmt.Sprintf("Bad date input '%s'. Please use the format: YYYY-MM-DD", dateInput)
		return content, false
	}

	newSSE := &SecretSantaEvent{
		Participants: map[string]Participant{},
	}
	newSSE.OrganizationDate = time.Now().String()
	caller, err := b.getUser(msg.Author)
	if err != nil {
		fmt.Println(err)
		return "", false
	}
	newSSE.Organizer = caller
	newSSE.DistributionDate = distributionDate.Format("2006-01-02")
	newSSE.SpendLimit = spendLimit

	content = fmt.Sprintf("%s is organizing a Secret Santa event! It will take place on %s.", newSSE.Organizer.Mention(), newSSE.DistributionDate)
	content += "\nTo join, react to this message!"
	content += fmt.Sprintf("\nPlease limit your spending according to: %s", newSSE.SpendLimit)
	content += fmt.Sprintf("\n%s, you can start the event whenever you're ready with '!start', so long as at least THREE participants have joined.", newSSE.Organizer.Mention())
	content += "\nOr, the event can be canceled with '!cancel'."

	b.Events[server.ID] = *newSSE

	return content, true
}

func (b *botStore) handleMsgStatus(msg *sgo.EventMessage) string {
	var content string

	server, err := b.getServerByChannelID(msg.Channel)
	if err != nil {
		return ""
	}
	sse, ok := b.Events[server.ID]
	if !ok {
		content = fmt.Sprintf("No Secret Santa events are currently active in the %s serer.", server.Name)
		content += "\nOne may be initiated with the '!new' command."
		return content
	}

	details := fmt.Sprintf("EVENT DETAILS:\n  - Distribution Date: %s\n  - Spending Limit: %s", sse.DistributionDate, sse.SpendLimit)
	if sse.hasStarted() {
		content = fmt.Sprintf("The Secret Santa event organized by %s has started, with %d participants!", sse.Organizer.Mention(), len(sse.Participants))
		_, ok := sse.Participants[msg.Author]
		if ok {
			caller, err := b.getUser(msg.Author)
			if err == nil {
				content += fmt.Sprintf("\nYou, %s, are one of them!", caller.Mention())
			}
		}
	} else {
		channelName := ""
		channel, err := b.getChannel(sse.JoinMessageChannelID)
		if err != nil {
			channelName = "???"
		} else {
			channelName = channel.Name
		}
		joinMessageLink := fmt.Sprintf("[join message](%s%s)", sgo.BaseURL(), sgo.EndpointChannelMessage(sse.JoinMessageChannelID, sse.JoinMessageID))
		content = fmt.Sprintf("A Secret Santa event organized by %s is active, and awaiting more participants.", sse.Organizer.Mention())
		content += fmt.Sprintf("\nNew participants may join by reacting to the %s I sent to the '%s' channel!", joinMessageLink, channelName)
	}
	content += "\n" + details

	return content
}

// handleMsgStart reads all unique reactions on the join message,
// and creates a set of participants for the Secret Santa event.
// There must be at least THREE participants for the event to start.
func (b *botStore) handleMsgStart(msg *sgo.EventMessage) string {
	server, err := b.getServerByChannelID(msg.Channel)
	if err != nil {
		return ""
	}
	sse, ok := b.Events[server.ID]
	if !ok {
		return "No Secret Santa events exist from this server to start."
	}

	// only the organizer may start the event
	if msg.Author != sse.Organizer.ID {
		return ""
	}

	joinMessage, err := b.session.ChannelMessage(sse.JoinMessageChannelID, sse.JoinMessageID)
	if err != nil {
		return "ERROR: could not find join message to read."
	}
	fmt.Println("Found joinMessage: " + joinMessage.Content)

	recorded := []string{}
	fmt.Printf("There are %d total reactions to evaluate!\n", len(joinMessage.Reactions))
	for r, uIDs := range joinMessage.Reactions {
		fmt.Println("Evaluating emoji reaction: " + r)
		for _, uID := range uIDs {
			fmt.Printf("uID %s reacted with %s...\n", uID, r)
			if exists := slices.Contains(recorded, uID); !exists {
				fmt.Printf("uID %s now being recorded as participant...\n", uID)
				recorded = append(recorded, uID)
			}
		}
	}

	if len(recorded) < 3 {
		content := "Uh oh! The Secret Santa event doesn't have enough participants. Use '!start' when there are at least three who have joined by reacting to the join message."
		content += fmt.Sprintf("\nParticipant signups: %d", len(recorded))
		return content
	}

	sse.assignParticipants(recorded)

	// NOTE: this call is for debugging purposes!
	sse.printParticipantMapping(b)

	b.Events[server.ID] = sse
	err = b.syncEventParticipants(server.ID)
	if err != nil {
		return "ERROR: could not sync event participants."
	}

	content := fmt.Sprintf("A Secret Santa event organized by %s has begun!", sse.Organizer.Mention())
	content += fmt.Sprintf("\n%d participants will be notified privately with next steps!", len(b.Events[server.ID].Participants))

		if err != nil {
		}

	return content
}

func (b *botStore) handleMsgCancel(msg *sgo.EventMessage) string {
	server, err := b.getServerByChannelID(msg.Channel)
	if err != nil {
		return ""
	}
	sse, ok := b.Events[server.ID]
	if !ok {
		return "No Secret Santa events exist from this server to cancel."
	}

	// only the organizer may start the event
	if msg.Author != sse.Organizer.ID {
		return ""
	}

	delete(b.Events, server.ID)
	return "Canceled existing Secret Santa event."
}

func (b *botStore) handleMsgHelp() string {
	return "Available commands: !help !new !start !cancel !ping"
}

func (b *botStore) handleMsgPing() string {
	latency := b.session.WS.Latency()

	if latency.Milliseconds() == 0 {
		return "Still calculating, keep re-trying this command in 15-second intervals."
	}

	return b.session.WS.Latency().String()
}
