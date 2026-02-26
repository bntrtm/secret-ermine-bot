package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

// getUser returns a user by ID, first trying to pull from cache
func (b *botStore) getUser(uID string) (*sgo.User, error) {
	var user *sgo.User
	user = b.session.State.User(uID)
	if user != nil {
		fmt.Printf("User %s fetched from cache\n", user.Username)
		return user, nil
	}
	user, err := b.session.User(uID)
	if err != nil {
		return nil, fmt.Errorf("User with ID %s could not be fetched", uID)
	}
	return user, nil
}

// getServer returns a server by ID, first trying to pull from cache
func (b *botStore) getServer(sID string) (*sgo.Server, error) {
	var server *sgo.Server
	server = b.session.State.Server(sID)
	if server != nil {
		fmt.Printf("Server %s fetched from cache\n", server.Name)
		return server, nil
	}
	server, err := b.session.Server(sID)
	if err != nil {
		return nil, fmt.Errorf("Server with ID %s could not be fetched", sID)
	}
	return server, nil
}

// getChannel returns a channel by ID, first trying to pull from cache
func (b *botStore) getChannel(cID string) (*sgo.Channel, error) {
	var channel *sgo.Channel
	channel = b.session.State.Channel(cID)
	if channel != nil {
		fmt.Printf("Channel %s fetched from cache\n", channel.Name)
		return channel, nil
	}
	channel, err := b.session.Channel(cID)
	if err != nil {
		return nil, fmt.Errorf("Channel with ID %s could not be fetched", cID)
	}
	return channel, nil
}

// getServerByChannelID wraps channel and server retrieval functions
// to JUST return the server, or nil with any error encountered
func (b *botStore) getServerByChannelID(cID string) (*sgo.Server, error) {
	channel, err := b.getChannel(cID)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	server, err := b.getServer(*channel.Server)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return server, nil
}

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

	if !strings.HasPrefix(msg.Content, "!") {
		return
	}

	fields := strings.Split(msg.Content, " ")
	command, args := strings.TrimPrefix(fields[0], "!"), fields[1:]
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
		content += "One may be initiated with the '!new' command."
		return content
	}

	details := fmt.Sprintf("EVENT DETAILS:\n  - Distribution Date: %s\n  - Spending Limit: %s", sse.DistributionDate, sse.SpendLimit)
	if len(sse.Participants) >= 3 {
		content = fmt.Sprintf("The Secret Santa event organized by %s has started, involving %d participants!", sse.Organizer.Mention(), len(sse.Participants))
		_, ok := sse.Participants[msg.Author]
		if ok {
			caller, err := b.getUser(msg.Author)
			if err == nil {
				content += fmt.Sprintf("\nYou, %s, are one of them!", caller.Mention())
			}
		}
		content += "\nOne may be initiated with the '!new' command."
		return content
	} else {
		joinMessageLink := fmt.Sprintf("[join message](%s)", sgo.EndpointChannelMessage(sse.JoinMessageChannelID, sse.JoinMessageID))
		content = fmt.Sprintf("A Secret Santa event organized by %s is active, and awaiting more participants.", sse.Organizer.Mention())
		content += fmt.Sprintf("\nNew participants may join by reacting to the %s I sent to the '%s' channel!", joinMessageLink, sse.JoinMessageChannelID)
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
	for r, uIDs := range joinMessage.Reactions {
		fmt.Println("Evaluating emoji reaction: " + r)
		for _, uID := range uIDs {
			if _, exists := sse.Participants[uID]; !exists {
				recorded = append(recorded, uID)
				sse.Participants[uID] = Participant{}
			}
		}
	}

	if len(recorded) < 3 {
		content := "Uh oh! The Secret Santa event doesn't have enough participants. Use '!start' when there are at least three who have joined by reacting to the join message."
		content += fmt.Sprintf("\nParticipants: %d", len(recorded))
		return content
	}

	// shuffle recorded set of participants
	for i := range recorded {
		j := rand.Intn(i + 1)
		recorded[i], recorded[j] = recorded[j], recorded[i]
	}

	for i, uID := range recorded {
		if pt, ok := sse.Participants[uID]; ok {
			gifteeIndex := i + 1
			// if I'm last, my giftee is the first participant
			if i == len(recorded)-1 {
				gifteeIndex = 0
			}
			pt.Giftee = recorded[gifteeIndex]
			// assuming they're a participant...
			if ptGiftee, ok := sse.Participants[pt.Giftee]; ok {
				// tell the system I'm their Secret Santa...
				ptGiftee.SecretSanta = uID
				sse.Participants[pt.Giftee] = ptGiftee
			}
			// ...and tell the system I know who my giftee is
			sse.Participants[uID] = pt
		}
	}

	b.Events[server.ID] = sse

	content := fmt.Sprintf("A Secret Santa event organized by %s has begun!", sse.Organizer.Mention())
	content += fmt.Sprintf("\n%d participants will be notified privately with next steps!", len(b.Events[server.ID].Participants))

	// NOTE: the following is temporary, and exists for debugging purposes!
	debugLine := "SSE RESULTS: "
	for uID, p := range b.Events[server.ID].Participants {
		user, err := b.getUser(uID)
		if err != nil {
			user = &sgo.User{
				ID:       "UNKNOWN",
				Username: "UNKNOWN",
			}
		}
		giftee, err := b.getUser(p.Giftee)
		if err != nil {
			giftee = &sgo.User{Username: p.Giftee}
		}
		santa, err := b.getUser(p.SecretSanta)
		if err != nil {
			santa = &sgo.User{Username: p.SecretSanta}
		}
		debugLine += fmt.Sprintf("\n%s is the Secret Santa of giftee: %s. Their Secret Santa is: %s", user.Username, giftee.Username, santa.Username)
	}
	fmt.Println(debugLine)

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
