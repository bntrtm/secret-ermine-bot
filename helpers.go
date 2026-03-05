package main

import (
	"fmt"
	"math/rand"
	"strings"

	sgo "github.com/sentinelb51/revoltgo"
)

// shuffleStrings shuffles a slice of strings in-place.
func shuffleStrings(strings []string) {
	for i := range strings {
		j := rand.Intn(i + 1)
		strings[i], strings[j] = strings[j], strings[i]
	}
}

func NewContext(session *sgo.Session, eventMsg *sgo.EventMessage) (*Context, error) {
	caller, err := getUser(session, eventMsg.Author)
	if err != nil {
		return nil, fmt.Errorf("could not recognize message author by ID %s: %w", eventMsg.Author, err)
	} else if caller == nil {
		return nil, fmt.Errorf("could not recognize message author by ID %s", eventMsg.Author)
	}

	channel, err := getChannel(session, eventMsg.Channel)
	if err != nil {
		return nil, fmt.Errorf("could not recognize channel: %w", err)
	} else if channel == nil {
		return nil, fmt.Errorf("could not recognize channel")
	}

	var server *sgo.Server
	if channel.ChannelType != sgo.ChannelTypeDM && channel.Server != nil {
		server, err = getServer(session, *channel.Server)
		if err != nil {
			return nil, fmt.Errorf("could not recognize server")
		}
	}

	return &Context{
		Session: session,
		Channel: channel,
		Server:  server,
		Caller:  caller,
		Message: &eventMsg.Message,
	}, nil
}

// validateCommandMessage parses a message, checking whether or not
// the bot ought to recognize it as a command call. The command is
// returned separated from the prefix, alongside all other space-separated
// substrings of the message as "args".
//
// The prefix used by the command caller is also returned, if valid
// for the channel.
func validateCommandMessage(ctx *Context) (prefix, command string, args []string, isValid bool) {
	fields := strings.Split(ctx.Message.Content, " ")

	// command calls to the bot in servers channels must first mention it
	self := ctx.Session.State.Self()
	permittedPrefixes := []string{fmt.Sprintf("%s !", self.Mention())}

	// commands for the bot from DM channels need not include the bot mention,
	// though it is still valid form
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		permittedPrefixes = append(permittedPrefixes, "!")
	}

	prefix = ""
	for _, p := range permittedPrefixes {
		if strings.HasPrefix(ctx.Message.Content, p) {
			prefix = p
			break
		}
	}

	switch prefix {
	case "":
		isValid = false
		return
	case "!":
		command, args = strings.TrimPrefix(fields[0], "!"), fields[1:]
	default:
		command, args = strings.TrimPrefix(fields[1], "!"), fields[2:]
	}
	isValid = true

	return
}

// getUser returns a user by ID, first trying to pull from cache
func getUser(session *sgo.Session, uID string) (user *sgo.User, err error) {
	if uID == "" {
		return nil, fmt.Errorf("input user ID is empty")
	}
	user = session.State.User(uID)
	if user != nil {
		fmt.Printf("User %s fetched from cache\n", user.Username)
		return user, nil
	}
	user, err = session.User(uID)
	if err != nil {
		return nil, fmt.Errorf("user with ID %s could not be fetched: %w", uID, err)
	}
	return user, nil
}

// getServer returns a server by ID, first trying to pull from cache
func getServer(session *sgo.Session, sID string) (server *sgo.Server, err error) {
	if sID == "" {
		return nil, fmt.Errorf("input server ID is empty")
	}
	server = session.State.Server(sID)
	if server != nil {
		fmt.Printf("Server %s fetched from cache\n", server.Name)
		return server, nil
	}
	server, err = session.Server(sID)
	if err != nil {
		return nil, fmt.Errorf("server with ID %s could not be fetched: %w", sID, err)
	}
	return server, nil
}

// getChannel returns a channel by ID, first trying to pull from cache
func getChannel(session *sgo.Session, cID string) (channel *sgo.Channel, err error) {
	if cID == "" {
		return nil, fmt.Errorf("input channel ID is empty")
	}
	channel = session.State.Channel(cID)
	if channel != nil {
		fmt.Printf("Channel %s fetched from cache\n", channel.Name)
		return channel, nil
	}
	channel, err = session.Channel(cID)
	if err != nil {
		return nil, fmt.Errorf("channel with ID %s could not be fetched: %w", cID, err)
	}
	return channel, nil
}

func makeEmbeddedMessage(title, description string) *sgo.MessageSend {
	embed := &sgo.MessageEmbed{}
	embed.Title = title
	embed.Description = description
	send := &sgo.MessageSend{
		Embeds: []*sgo.MessageEmbed{
			embed,
		},
	}
	return send
}
