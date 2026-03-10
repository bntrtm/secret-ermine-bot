package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

func (b *botStore) handlerEventMessage(ctx *Context) {
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

		botMsg, err := ctx.Session.ChannelMessageSend(ctx.Channel.ID, send)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}

		fmt.Println("Sent message: ", botMsg.Content)

		if recordJoinMessage && ctx.Channel.ChannelType != sgo.ChannelTypeDM {
			if entry, ok := b.Events[ctx.Server.ID]; ok {
				entry.JoinMessageChannelID = botMsg.Channel
				entry.JoinMessageID = botMsg.ID
				b.Events[ctx.Server.ID] = entry
			}
		}
	}()

	prefix, command, args, isValid := validateCommandMessage(ctx, getValidPrefixes(ctx))
	if !isValid {
		return
	}

	switch command {
	case "new":
		success := false
		content, success = b.handleMsgNew(args, ctx)
		recordJoinMessage = success
	case "start":
		content = b.handleMsgStart(ctx)
	case "status":
		content = b.handleMsgStatus(ctx)
	case "help":
		content = b.handleMsgHelp()
	case "ping":
		content = b.handleMsgPing(ctx)
	case "cancel":
		content = b.handleMsgCancel(ctx)
	case "dearsanta":
		content = b.handleDearParticipant(ctx, Santa, strings.TrimPrefix(ctx.Message.Content, prefix+command))
	case "deargiftee":
		content = b.handleDearParticipant(ctx, Giftee, strings.TrimPrefix(ctx.Message.Content, prefix+command))
	default:
		content = fmt.Sprintf("Unknown command '%s', use '!help' for all available commands.", "!"+command)
	}
}

// handleMsgNew tells the bot it's time for a new Secret Santa Session!
// usage: !new <date (YYYY-MM-DD)> <spend_limit>
func (b *botStore) handleMsgNew(args []string, ctx *Context) (string, bool) {
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		return "Secret Santa events must be started within a server channel.", false
	}

	if event, ok := b.Events[ctx.Server.ID]; ok {
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
	caller, err := getUser(ctx.Session, ctx.Caller.ID)
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

	b.Events[ctx.Server.ID] = *newSSE

	return content, true
}

// TODO: implement a way for users to specify WHICH secret santa event
// they wish to refer to when writing a command, WHEN they are in more
// than one managed by the same bot instance.

func (b *botStore) handleMsgStatus(ctx *Context) string {
	var content string
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		sID, matches, err := b.getParticipantEvent(ctx.Caller.ID, "")
		if err != nil {
			if matches == 0 {
				content = "You are not a participant in any Secret Santa events that I'm managing."
				return content
			} else if matches > 1 {
				content = fmt.Sprintf("You were found as a participant in %d Secret Santa events; I don't know which you want a status report on!", matches)
				return content
			}
		}
		sse, ok := b.Events[sID]
		if !ok {
			fmt.Printf("could not find event identified by sID '%s'\n", sID)
			return ""
		}
		server, err := getServer(ctx.Session, sID)
		if err != nil {
			fmt.Printf("could not get server by id '%s': %s\n", sID, err)
			return ""
		}
		content = fmt.Sprintf("You are a participant in a Secret Santa event from the %s server, organized by %s!", server.Name, sse.Organizer.Mention())
		return content
	}

	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		content = fmt.Sprintf("No Secret Santa events are currently active in the %s server.", ctx.Server.Name)
		content += "\nOne may be initiated with the '!new' command."
		return content
	}

	if sse.hasStarted() {
		content = fmt.Sprintf("The Secret Santa event organized by %s has started, with %d participants!", sse.Organizer.Mention(), len(sse.Participants))
		_, ok := sse.Participants[ctx.Caller.ID]
		if ok {
			content += fmt.Sprintf("\nYou, %s, are one of them!", ctx.Caller.Mention())
		}
	} else {
		channelName := ""
		channel, err := getChannel(ctx.Session, sse.JoinMessageChannelID)
		if err != nil {
			channelName = "???"
		} else {
			channelName = channel.Name
		}
		joinMessageLink := fmt.Sprintf("[join message](%s%s)", sgo.BaseURL(), sgo.EndpointChannelMessage(sse.JoinMessageChannelID, sse.JoinMessageID))
		content = fmt.Sprintf("A Secret Santa event organized by %s is active, and awaiting more participants.", sse.Organizer.Mention())
		content += fmt.Sprintf("\nNew participants may join by reacting to the %s I sent to the '%s' channel!", joinMessageLink, channelName)
	}
	content += "\n" + sse.details()

	return content
}

// handleMsgStart reads all unique reactions on the join message,
// and creates a set of participants for the Secret Santa event.
// There must be at least THREE participants for the event to start.
func (b *botStore) handleMsgStart(ctx *Context) string {
	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		return "No Secret Santa events exist from this server to start."
	}

	// only the organizer may start the event
	if ctx.Caller.ID != sse.Organizer.ID {
		return ""
	}

	joinMessage, err := ctx.Session.ChannelMessage(sse.JoinMessageChannelID, sse.JoinMessageID)
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
	sse.printParticipantMapping(ctx.Session)

	b.Events[ctx.Server.ID] = sse
	err = b.syncEventParticipants(ctx.Server.ID)
	if err != nil {
		return "ERROR: could not sync event participants."
	}

	content := fmt.Sprintf("A Secret Santa event organized by %s has begun!", sse.Organizer.Mention())
	content += fmt.Sprintf("\n%d participants will be notified privately with next steps!", len(b.Events[ctx.Server.ID].Participants))

	go func() {
		err = b.notifySantas(ctx)
		if err != nil {
			fmt.Println("notifySantas: %w", err)
		}
	}()

	return content
}

func (b *botStore) handleMsgCancel(ctx *Context) string {
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		return "Event organizers may only cancel events from the server in which they were started, through any channel."
	}

	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		return "No Secret Santa events exist from this server to cancel."
	}

	// only the organizer may cancel the event
	if ctx.Caller.ID != sse.Organizer.ID {
		return ""
	}

	delete(b.Events, ctx.Server.ID)
	b.cleanTrackedParticipants()
	return "Canceled existing Secret Santa event."
}

func (b *botStore) handleDearParticipant(ctx *Context, subject ParticipantRelation, letterContent string) (content string) {
	if ctx.Channel.ChannelType != sgo.ChannelTypeDM {
		return
	}
	sID, matches, err := b.getParticipantEvent(ctx.Caller.ID, "")
	if err != nil {
		if matches == 0 {
			return
		} else if matches > 1 {
			content = fmt.Sprintf("You were found as a participant in %d Secret Santa events; I don't know which of your %ss you want to write to!", matches, subject.Title())
			return content
		}
	}

	sse, ok := b.Events[sID]
	if !ok {
		content = "You are not a participant in any Secret Santa events that I'm managing."
		return
	} else if !sse.hasStarted() {
		content = "The Secret Santa event has not started yet!"
		return
	}

	errorMessageContent := fmt.Sprintf("Sorry, I was unable to send the message to your %s.", subject.Title())

	var subjectUID string
	switch subject {
	case Santa:
		subjectUID = sse.Participants[ctx.Caller.ID].SecretSanta
	case Giftee:
		subjectUID = sse.Participants[ctx.Caller.ID].Giftee
	default:
		content = fmt.Sprintf("Sorry, I was unable to send the message to your %s.", subject.Title())
		return
	}

	var messageToSubject string
	switch subject {
	case Santa:
		messageToSubject = fmt.Sprintf("Dear Santa,\n%s\nSincerely, %s", letterContent, ctx.Caller.Mention())
	case Giftee:
		subjectUser, err := getUser(ctx.Session, subjectUID)
		if err != nil {
			fmt.Printf("could not get %s assigned to caller with username %s: %s\n", subject.Title(), ctx.Caller.Username, err)
			content = errorMessageContent
			return
		}

		messageToSubject = fmt.Sprintf("Dear %s,\n%s\nSincerely, Santa", subjectUser.Mention(), letterContent)
	}

	send := makeEmbeddedMessage(ColourSoftRed, fmt.Sprintf("**You received a letter from your %s!**", subject.Opp().Title()), messageToSubject)

	err = b.sendDM(ctx.Session, send, subjectUID)
	if err != nil {
		fmt.Printf("failed to message a santa on behalf of user: %s\n", ctx.Caller.Mention())
		content = errorMessageContent
		return
	}
	content = fmt.Sprintf("I've sent your message to your %s!", subject.Title())
	return
}

func (b *botStore) handleMsgHelp() string {
	return "Available commands:\n-!help\n-!new\n-!start\n-!status\n-!cancel\n-!ping"
}

func (b *botStore) handleMsgPing(ctx *Context) string {
	latency := ctx.Session.WS.Latency()

	if latency.Milliseconds() == 0 {
		return "Still calculating, keep re-trying this command in 15-second intervals."
	}

	return ctx.Session.WS.Latency().String()
}
