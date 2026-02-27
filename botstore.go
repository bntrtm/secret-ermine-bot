package main

import (
	"fmt"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

// botStore tracks persistent data related to the bot's activity across one or more servers
type botStore struct {
	session *sgo.Session

	Token  string
	Events map[string]SecretSantaEvent // map of servers to secret-santa events (limited to one active SSE/server)
}

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
