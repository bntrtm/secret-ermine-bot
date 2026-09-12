package main

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	// 'sgo' as in "stoat go"

	"github.com/bntrtm/secret-ermine-bot/model"
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
				b.saveEvent(ctx.Server.ID, entry)
			}
		}
	}()

	prefix, command, args, isValid := validateCommandMessage(ctx, getValidPrefixes(ctx))
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
		content, err = b.handleMsgStatus(ctx)
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
		content = b.handleDearParticipant(ctx, Santa, strings.TrimPrefix(ctx.Message.Content, prefix+command))
	case "dear-giftee":
		content = b.handleDearParticipant(ctx, Giftee, strings.TrimPrefix(ctx.Message.Content, prefix+command))
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
		organizer, err := getEventOrganizer(ctx.Session, event)
		if err != nil {
			return "A Secret Santa event is already active in this server.\nThe organizer must use the '!cancel' command before setting up a new one.", fmt.Errorf("event for this server already active")
		}

		return fmt.Sprintf("A Secret Santa event organized by %s is already active in this server.\nThey must use the '!cancel' command before setting up a new one.", organizer.Mention()), fmt.Errorf("event for this server already active")
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

	newSSE := &model.SecretSantaEvent{
		Participants: map[string]model.Participant{},
	}
	newSSE.OrganizationDate = time.Now().String()
	caller, err := getUser(ctx.Session, ctx.Caller.ID)
	if err != nil {
		return "", err
	}
	newSSE.OrganizerID = caller.ID
	newSSE.DistributionDate = distributionDate.Format("2006-01-02")
	newSSE.Notes = strings.Join(notes, " ")

	if err := b.saveEvent(ctx.Server.ID, *newSSE); err != nil {
		content = "I'm sorry, I couldn't get the event started. Try again later."
		return content, fmt.Errorf("could not save event to database: %w", err)
	}

	organizer, err := getEventOrganizer(ctx.Session, *newSSE)
	if err != nil {
		return "", fmt.Errorf("could not get organizer from event: %v", err)
	}

	content = fmt.Sprintf("%s is organizing a Secret Santa event! It will take place on %s.", organizer.Mention(), newSSE.DistributionDate)
	content += " **To join, react to this message!**"
	if newSSE.Notes != "" {
		content += " Organizer's notes regarding the event:"
		content += "\n\"" + newSSE.Notes + "\""
	}

	return content, nil
}

func (b *botStore) saveEvent(serverID string, event model.SecretSantaEvent) error {
	err := b.repo.SaveEvent(serverID, event)
	if err != nil {
		return fmt.Errorf("could not save event")
	}

	b.Events[serverID] = event

	return nil
}

// TODO: implement a way for users to specify WHICH secret santa event
// they wish to refer to when writing a command, WHEN they are in more
// than one managed by the same bot instance.

func (b *botStore) handleMsgStatus(ctx *Context) (string, error) {
	var content string
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		sID, matches, err := b.getParticipantEvent(ctx.Caller.ID, "")
		if err != nil {
			if matches == 0 {
				content = "You are not a participant in any Secret Santa events that I'm managing."
				return content, nil
			} else if matches > 1 {
				content = fmt.Sprintf("You were found as a participant in %d Secret Santa events; I don't know which you want a status report on!", matches)
				return content, nil
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

		organizer, err := getEventOrganizer(ctx.Session, sse)
		if err != nil {
			return "", fmt.Errorf("could not get organizer from event: %v", err)
		}

		content = fmt.Sprintf("You are a participant in a Secret Santa event from the %s server, organized by %s!", server.Name, organizer.Mention())
		return content, nil
	}

	sse, ok := b.Events[ctx.Server.ID]
	if !ok {
		content = fmt.Sprintf("No Secret Santa events are currently active in the %s server.", ctx.Server.Name)
		content += "\nOne may be initiated with the '!new' command."
		return content, nil
	}

	organizer, err := getEventOrganizer(ctx.Session, sse)
	if err != nil {
		return "", fmt.Errorf("could not get organizer from event: %v", err)
	}

	if sse.HasStarted() {
		content = fmt.Sprintf("The Secret Santa event organized by %s has started, with %d participants!", organizer.Mention(), len(sse.Participants))
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

		organizer, err := getEventOrganizer(ctx.Session, sse)
		if err != nil {
			return "", fmt.Errorf("could not get organizer from event: %v", err)
		}

		joinMessageLink := singularizeStoatRoute(fmt.Sprintf("[join message](%s%s%s/%s)", strings.TrimRight(b.baseUrl, "/"), sgo.EndpointServer(ctx.Server.ID), sgo.EndpointChannel(sse.JoinMessageChannelID), sse.JoinMessageID))
		content = fmt.Sprintf("A Secret Santa event organized by %s is active, and awaiting more participants.", organizer.Mention())
		content += fmt.Sprintf("\nNew participants may join by reacting to the join message I sent to the '%s' channel!", channelName)
		// Stoat renders a message link without the hyprlink text.
		content += fmt.Sprintf("\nJOIN HERE: %s", joinMessageLink)
	}
	content += "\n" + sse.Details()

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
	if ctx.Caller.ID != sse.OrganizerID {
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

	sse.AssignParticipants(recorded)

	{

		// NOTE: this call is for debugging purposes!
		getName := func(uID string) string {
			if ctx.Session != nil {
				user, err := getUser(ctx.Session, uID)
				if err != nil {
					return uID
				}
				return user.Username
			} else {
				return uID
			}
		}

		sse.PrintParticipantMapping(getName)
	}

	b.Events[ctx.Server.ID] = sse
	err = b.syncEventParticipants(ctx.Server.ID)
	if err != nil {
		return "ERROR: could not sync event participants."
	}

	organizer, err := getEventOrganizer(ctx.Session, sse)
	if err != nil {
		return fmt.Sprintf("could not get organizer from event: %v", err)
	}

	content := fmt.Sprintf("A Secret Santa event organized by %s has begun!", organizer.Mention())
	content += fmt.Sprintf("\n%d participants will be notified privately with next steps!", len(b.Events[ctx.Server.ID].Participants))

	go func() {
		err = b.notifySantas(ctx)
		if err != nil {
			b.TryServerLogError(ctx, "notifySantas: "+err.Error())
		}
	}()

	return content
}

func (b *botStore) cancelEvent(serverID string) error {
	if err := b.repo.DeleteEvent(serverID); err != nil {
		return err
	}
	delete(b.Events, serverID)
	b.cleanTrackedParticipants()
	return nil
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
	if ctx.Caller.ID != sse.OrganizerID {
		return ""
	}

	err := b.cancelEvent(ctx.Server.ID)
	if err != nil {
		return "Sorry, I couldn't cancel the event. Try again later."
	}

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
	} else if !sse.HasStarted() {
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
			b.logger.ELogError(sID, fmt.Sprintf("could not get %s assigned to caller with username %s: %s\n", subject.Title(), ctx.Caller.Username, err))
			content = errorMessageContent
			return
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
