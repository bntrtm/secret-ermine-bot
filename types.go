package main

import (
	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

// These constants may be used to points users to a public repository
// for the source code of this bot.
const (
	BotName           = "Secret Ermine"
	BotAvatarURL      = "https://raw.githubusercontent.com/bntrtm/secret-ermine-bot/b923e63c083e97fcaf66e85dcb650eff752e827b/assets/Secret_Ermine_Bot_Icon.svg"
	BotSourceCodeLink = "https://github.com/bntrtm/secret-ermine-bot"
)

// command represents an action that a user may request
// of the bot. Here, it is simply used in the process of
// building the help message.
type command struct {
	name                  string
	description           string
	dmChannelsEnabled     bool
	serverChannelsEnabled bool
	devOnly               bool
}

// Context represents a single source of truth
// retaining all relevant details about a message event.
type Context struct {
	Session *sgo.Session
	Channel *sgo.Channel
	Server  *sgo.Server
	Caller  *sgo.User
	Message *sgo.Message
}

type ParticipantRelation int

const (
	Santa ParticipantRelation = iota
	Giftee
)

// Opp returns the corresponding value of the participant relation.
func (r *ParticipantRelation) Opp() ParticipantRelation {
	if *r == Santa {
		return Giftee
	}
	return Santa
}

// Title returns the formatted human-readable relation type
// for use in output or messages
func (r ParticipantRelation) Title() string {
	switch r {
	case Santa:
		return "Secret Santa"
	case Giftee:
		return "giftee"
	default:
		return "<UNKNOWN PARTICIPANT RELATION>"
	}
}

const ColourSoftRed = "#FF3939"
