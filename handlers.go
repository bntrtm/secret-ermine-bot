package main

import (
	"fmt"
	"log/slog"
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
			Content:    content,
			Masquerade: b.Masquerade,
		}

		botMsg, err := ctx.Session.ChannelMessageSend(ctx.Channel.ID, send)
		if err != nil {
			b.TryServerLogError(ctx, fmt.Sprintf("could not send message; %s", err), slog.String("message", send.Content))
			return
		}

		if ctx.Server != nil {
			b.TryServerLogInfo(ctx, "sent message to server", slog.String("message", botMsg.Content))
		}

		if recordJoinMessage && ctx.Channel.ChannelType != sgo.ChannelTypeDM {
			if entry, ok := b.Events[ctx.Server.ID]; ok {
				entry.JoinMessageChannelID = botMsg.Channel
				entry.JoinMessageID = botMsg.ID
				b.Events[ctx.Server.ID] = entry
			}
		}
	}()

	_, command, args, isValid := validateCommandMessage(ctx, getValidPrefixes(ctx))
	if !isValid {
		return
	}

	if cmd, ok := b.commands[command]; ok && cmd.devOnly && b.platform != "DEV" {
		content = fmt.Sprintf("Invalid command '%s', use *!help* for all available commands.", command)
		return
	}

	switch command {
	case "new":
		var err error
		content, err = b.handleMsgNew(args, ctx)
		if err == nil {
			recordJoinMessage = true
		} else {
			b.TryServerLogError(ctx, err.Error())
		}
	case "start":
		content = b.handleMsgStart(ctx)
	case "status":
		var err error
		sIDPrefix, _ := trimServerIDArg(args)
		content, err = b.handleMsgStatus(ctx, sIDPrefix)
		if err != nil {
			b.TryServerLogError(ctx, err.Error())
		}
	case "help":
		content = b.handleMsgHelp(ctx)
	case "ping":
		content = b.handleMsgPing(ctx)
	case "cancel":
		content = b.handleMsgCancel(ctx)
	case "dear-santa":
		sIDPrefix, args := trimServerIDArg(args)
		content = b.handleDearParticipant(ctx, sIDPrefix, Santa, strings.Join(args, " "))
	case "dear-giftee":
		sIDPrefix, args := trimServerIDArg(args)
		content = b.handleDearParticipant(ctx, sIDPrefix, Giftee, strings.Join(args, " "))
	case "about":
		if BotName == "" {
			content = ""
		} else if b.AboutLinkParsed {
			content = fmt.Sprintf("I'm an instance of the [%s bot](%s)!", BotName, BotSourceCodeLink)
		} else {
			content = fmt.Sprintf("I'm an instance of the %s bot!", BotName)
		}
	default:
		content = fmt.Sprintf("Unknown command '%s', use *!help* for all available commands.", command)
	}
}

// handleMsgNew tells the bot it's time for a new Secret Santa Session!
// usage: !new <date (YYYY-MM-DD)> <notes>
func (b *botStore) handleMsgNew(args []string, ctx *Context) (string, error) {
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		return "Secret Santa events must be started within a server channel.", fmt.Errorf("event must be started outside of direct message channel")
	}

	if event, ok := b.Events[ctx.Server.ID]; ok {
		return fmt.Sprintf("A Secret Santa event organized by %s is already active in this server.\nThey must use the '!cancel' command before setting up a new one.", event.Organizer.Mention()), fmt.Errorf("event for this server already active")
	}

	var content string

	if len(args) == 0 {
		content = "Missing one or more arguments. Use the *!help* command for further information."
		return content, fmt.Errorf("argument mismatch; expected at least 1, but got %d", len(args))
	}

	notes := []string{}
	dateInput := args[0]
	if len(args) > 1 {
		notes = args[1:]
	}

	distributionDate, err := time.Parse("2006-01-02", dateInput)
	if err != nil {
		content = fmt.Sprintf("Bad date input '%s'. Please use the format: YYYY-MM-DD", dateInput)
		return content, fmt.Errorf("could not parse distribution date provided as time.Time: %w", err)
	}
	if distributionDate.Before(time.Now()) {
		content = fmt.Sprintf("Bad date input '%s'. Please set a future date for gift distribution.", dateInput)
		return content, fmt.Errorf("bad date input; must be future date")
	}

	newSSE := &SecretSantaEvent{
		Participants: map[string]Participant{},
	}
	newSSE.OrganizationDate = time.Now().String()
	caller, err := getUser(ctx.Session, ctx.Caller.ID)
	if err != nil {
		return "", err
	}
	newSSE.Organizer = caller
	newSSE.DistributionDate = distributionDate.Format("2006-01-02")
	newSSE.Notes = strings.Join(notes, " ")

	content = fmt.Sprintf("%s is organizing a Secret Santa event! It will take place on %s.", newSSE.Organizer.Mention(), newSSE.DistributionDate)
	content += " **To join, react to this message!**"
	if newSSE.Notes != "" {
		content += " Organizer's notes regarding the event:"
		content += "\n\"" + newSSE.Notes + "\""
	}

	b.Events[ctx.Server.ID] = *newSSE

	return content, nil
}

// makeNotifyTooManyEventsMessage wraps listParticipantEventsMessage
// create a helpful message informing a user that using a command will
// expect a serverID prefix be provided to the bot, due to their being
// a participant in two or more events hosted across different Stoat
// servers.
func makeNotifyTooManyEventsMessage(session *sgo.Session, sIDString string, numMatches int) (*sgo.MessageSend, error) {
	content := fmt.Sprintf("Because you were found as a participant in %d Secret Santa events, I don't know which you want a status report on!", numMatches)
	content += "\n\nTo let me know which event context to use, use the following command syntax: "
	content += "\n`!<command> --<server ID prefix>`"

	send, err := listParticipantEventsMessage(session, sIDString)
	if err != nil {
		return nil, fmt.Errorf("sendNotifyTooManyEventsMessage: %w", err)
	}

	send.Content = content

	return send, nil
}

// listParticipantEventsMessage creates an embedded message that lists
// the Secret Santa events that a user is participating in. It is meant
// to be called when the `getParticipantEvent` function would return
// more than one event, using the comma-separated string of server IDs
// to populate the embedded list.
func listParticipantEventsMessage(session *sgo.Session, sIDString string) (*sgo.MessageSend, error) {
	var description strings.Builder
	description.WriteString("You are a participant in the following events:\n")
	for sID := range strings.SplitSeq(sIDString, ",") {
		server, err := getServer(session, sID)
		if err != nil {
			return nil, fmt.Errorf("could not populate participant events list: %d", err)
		}
		fmt.Fprintf(&description, "\nSERVER EVENT: NAME - %s, ID - %s", server.Name, sID[:6])
	}
	message := makeEmbeddedMessage(ColourSoftRed, "Events You're In", description.String())

	return message, nil
}

func (b *botStore) handleMsgStatus(ctx *Context, sIDPrefix string) (string, error) {
	var content string
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		sID, matches, err := b.getParticipantEvent(ctx.Caller.ID, sIDPrefix)
		if err != nil {
			if matches == 0 {
				content = "You are not a participant in any Secret Santa events that I'm managing."
				return content, nil
			} else if matches > 1 {
				send, err := makeNotifyTooManyEventsMessage(ctx.Session, sID, matches)
				if err != nil {
					return "", err
				}
				_ = b.sendDM(ctx.Session, send, ctx.Caller.ID)
				return "", nil
			}
		}
		sse, ok := b.Events[sID]
		if !ok {
			return "", fmt.Errorf("could not find event identified by sID '%s'", sID)
		}
		server, err := getServer(ctx.Session, sID)
		if err != nil {
			return "", fmt.Errorf("could not get server by id '%s': %w", sID, err)
		}
		content = fmt.Sprintf("You are a participant in a Secret Santa event from the %s server, organized by %s!", server.Name, sse.Organizer.Mention())
		return content, nil
	}

	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		content = fmt.Sprintf("No Secret Santa events are currently active in the %s server.", ctx.Server.Name)
		content += "\nOne may be initiated with the '!new' command."
		return content, nil
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

	return content, nil
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
	b.TryServerLogDebug(ctx, "found join message for event", slog.String("content", joinMessage.Content))

	recorded := []string{}
	b.TryServerLogDebug(ctx, fmt.Sprintf("There are %d total reactions to evaluate!\n", len(joinMessage.Reactions)), slog.String("content", joinMessage.Content))
	for rID, uIDs := range joinMessage.Reactions {
		b.TryServerLogDebug(ctx, "Evaluating emoji reaction stack with ID: "+rID)
		for _, uID := range uIDs {
			b.TryServerLogDebug(ctx, fmt.Sprintf("user with ID %s reacted with %s...\n", uID, rID))
			if exists := slices.Contains(recorded, uID); !exists {
				b.TryServerLogDebug(ctx, fmt.Sprintf("user with ID %s now being recorded as participant...\n", uID))
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
			b.TryServerLogError(ctx, "notifySantas: "+err.Error())
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

func (b *botStore) handleDearParticipant(ctx *Context, sIDPrefix string, subject ParticipantRelation, letterContent string) (content string) {
	if ctx.Channel.ChannelType != sgo.ChannelTypeDM {
		return
	}
	sID, matches, err := b.getParticipantEvent(ctx.Caller.ID, sIDPrefix)
	if err != nil {
		if matches == 0 {
			content = "You are not a participant in any Secret Santa events that I'm managing."
			return content
		} else if matches > 1 {
			send, err := makeNotifyTooManyEventsMessage(ctx.Session, sID, matches)
			if err != nil {
				return
			}
			_ = b.sendDM(ctx.Session, send, ctx.Caller.ID)
			return
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
		return content
	}

	var messageToSubject string
	switch subject {
	case Santa:
		messageToSubject = fmt.Sprintf("Dear Santa,\n%s\nSincerely, %s", letterContent, ctx.Caller.Mention())
	case Giftee:
		subjectUser, err := getUser(ctx.Session, subjectUID)
		if err != nil {
			b.logger.ELogError(sID, fmt.Sprintf("could not get %s assigned to caller with username %s: %s\n", subject.Title(), ctx.Caller.Username, err))
			content = errorMessageContent
			return content
		}

		messageToSubject = fmt.Sprintf("Dear %s,\n%s\nSincerely, Santa", subjectUser.Mention(), letterContent)
	}

	send := makeEmbeddedMessage(ColourSoftRed, fmt.Sprintf("**You received a letter from your %s!**", subject.Opp().Title()), messageToSubject)

	err = b.sendDM(ctx.Session, send, subjectUID)
	if err != nil {
		b.logger.ELogError(sID, fmt.Sprintf("failed to message a santa on behalf of user: %s\n", ctx.Caller.Mention()))
		content = errorMessageContent
		return
	}
	content = fmt.Sprintf("I've sent your message to your %s!", subject.Title())
	return
}

func (b *botStore) handleMsgHelp(ctx *Context) string {
	var helpStr strings.Builder
	helpStr.WriteString("To write a command for the bot, use: !erm <command>")
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		helpStr.WriteString("\nHere in DMs with me, you may use this shorthand: !<command>")
	}
	helpStr.WriteString("\n\n**Available commands:**")
	for _, cmd := range b.commandKeys {
		info, ok := b.commands[cmd]
		if !ok {
			continue
		}
		if info.devOnly && b.platform != "DEV" {
			continue
		}

		if (info.dmChannelsEnabled && ctx.Channel.ChannelType == sgo.ChannelTypeDM) ||
			(info.serverChannelsEnabled && ctx.Channel.ChannelType != sgo.ChannelTypeDM) {
			fmt.Fprintf(&helpStr, "\n*%s:* %s", info.name, info.description)
		}
	}
	return helpStr.String()
}

func (b *botStore) handleMsgPing(ctx *Context) string {
	latency := ctx.Session.WS.Latency()

	if latency.Milliseconds() == 0 {
		return "Still calculating, keep re-trying this command in 15-second intervals."
	}

	return ctx.Session.WS.Latency().String()
}
