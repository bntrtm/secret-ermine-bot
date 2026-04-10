package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	// 'sgo' as in "stoat go"

	sgo "github.com/sentinelb51/revoltgo"
)

// shuffleStrings shuffles a slice of strings in-place.
func shuffleStrings(strings []string) {
	for i := range strings {
		j := rand.Intn(i + 1)
		strings[i], strings[j] = strings[j], strings[i]
	}
}

// getValidPrefixes evaluates prefixes that the bot ought
// to recognize as valid form in messages before caring
// to further parse those messages.
func getValidPrefixes(ctx *Context) []string {
	// NOTE: this logic was once simply included within the validateCommandMessage function.
	// After all, the necessary values can be pulled from context there just as well.
	// It has been separated simply for the sake of making that function testable as a unit.

	// command calls to the bot in server channels MUST use the !erm prefix
	prefixes := []string{"!erm "}

	// commands for the bot from DM channels need not include the !erm,
	// though it is still valid form
	if ctx.Channel.ChannelType == sgo.ChannelTypeDM {
		prefixes = append(prefixes, "!")
	}

	return prefixes
}

// validateCommandMessage parses a message, checking whether or not
// the bot ought to recognize it as a command call. The command is
// returned separated from the prefix, alongside all other space-separated
// substrings of the message as "args".
//
// The prefix used by the command caller is also returned, if valid
// for the channel.
func validateCommandMessage(ctx *Context, validPrefixes []string) (prefix, command string, args []string, isValid bool) {
	fields := strings.Fields(ctx.Message.Content)

	if len(validPrefixes) == 0 {
		return
	}

	prefix = ""
	for _, p := range validPrefixes {
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
		command, args = fields[1], fields[2:]
	}
	isValid = true

	return
}

// trimServerIDPrefixArg checks for a first argument that may be present
// to specify a server ID to the bot, which is required in some DM
// contexts. If it exists, it is stripped from the input slice of arguments
// and returned alongside the new modified slice.
func trimServerIDArg(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}
	if !strings.HasPrefix(args[0], "--") {
		return "", args
	}

	if len(args) == 1 {
		return strings.TrimLeft(args[0], "-"), []string{}
	}
	return strings.TrimLeft(args[0], "-"), args[1:]
}

// makeEmbeddedMessage produces a message with the given title
// and description text, colored a soft red.
func makeEmbeddedMessage(colour, title, description string) *sgo.MessageSend {
	embed := &sgo.MessageEmbed{}
	// in the Stoat API, the CSS format honors the regex check
	embed.Colour = colour
	embed.Title = title
	embed.Description = description
	send := &sgo.MessageSend{
		Embeds: []*sgo.MessageEmbed{
			embed,
		},
	}
	return send
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

// validateURL returns whether or not a string successfully parses as a URL.
func validateURL(urlString string) bool {
	_, err := url.Parse(urlString)
	return err == nil
}

// -----------------
// 	GETTERS
// -----------------

// getUser returns a user by ID, first trying to pull from cache
func getUser(session *sgo.Session, uID string) (user *sgo.User, err error) {
	if uID == "" {
		return nil, fmt.Errorf("input user ID is empty")
	}
	user = session.State.User(uID)
	if user != nil {
		// slog.Debug(fmt.Sprintf("User %s fetched from cache\n", user.Username))
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
		// slog.Debug(fmt.Sprintf("Server %s fetched from cache\n", server.Name))
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
		// slog.Debug(fmt.Sprintf("Channel %s fetched from cache\n", channel.Name))
		return channel, nil
	}
	channel, err = session.Channel(cID)
	if err != nil {
		return nil, fmt.Errorf("channel with ID %s could not be fetched: %w", cID, err)
	}
	return channel, nil
}
