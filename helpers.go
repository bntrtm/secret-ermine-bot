package main

import (
	"fmt"
	"math/rand"

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

// getUser returns a user by ID, first trying to pull from cache
func getUser(session *sgo.Session, uID string) (user *sgo.User, err error) {
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
