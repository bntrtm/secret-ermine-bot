package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bntrtm/secret-ermine-bot/internal/logging"
	"github.com/joho/godotenv"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

func getBotName() string {
	name := os.Getenv("MASQ_NAME")
	if name == "" {
		return BotName
	}
	return name
}

func getBotAvatarURL() string {
	url := os.Getenv("MASQ_AVATAR_URL")
	if strings.ToUpper(url) == "DISABLE" {
		return ""
	}
	return BotAvatarURL
}

func main() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	bot := &botStore{
		Events:              map[string]SecretSantaEvent{},
		TrackedParticipants: map[string]map[string]struct{}{},
		Token:               os.Getenv("BOT_TOKEN"),
		AboutLinkParsed:     validateURL(BotSourceCodeLink),
		Masquerade: &sgo.MessageMasquerade{
			Name:   getBotName(),
			Avatar: getBotAvatarURL(),
		},
		commands: []command{
			{
				name:                  "help",
				description:           "Get help regarding bot usage.",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: true,
			},
			{
				name:                  "new",
				description:           "Set up a new Secret Santa event in this server. Arguments: <Distribution Date *(YYYY-MM-DD)*> <Optional Notes... *(any text)*>",
				dmChannelsEnabled:     false,
				serverChannelsEnabled: true,
			},
			{
				name:                  "start",
				description:           "Start a Secret Santa event, so long as three unique participants have offered a reaction to the join message.",
				dmChannelsEnabled:     false,
				serverChannelsEnabled: true,
			},
			{
				name:                  "status",
				description:           "See the details of an existing Secret Santa event (or lack thereof) within this server.",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: true,
			},
			{
				name:                  "cancel",
				description:           "Cancel an existing Secret Santa event in this server.",
				dmChannelsEnabled:     false,
				serverChannelsEnabled: true,
			},
			{
				name:                  "dearsanta",
				description:           "Send a letter to your Secret Santa! Just follow it with the message you want to send!",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: false,
			},
			{
				name:                  "deargiftee",
				description:           "Send a letter to your giftee! Just follow it with the message you want to send!",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: false,
			},
			{
				name:                  "ping",
				description:           "Check websocket latency with this bot.",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: true,
			},
			{
				name:                  "about",
				description:           "Get info about this bot instance",
				dmChannelsEnabled:     true,
				serverChannelsEnabled: true,
			},
		},
	}

	// structured logging setup
	bot.logger = &logging.Logger{}
	bot.logger.Init()
	defer bot.logger.Quit()

	// start a new sgo session
	session := sgo.New(bot.Token)

	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventReady) {
		readyLogMessage := fmt.Sprintf("Ready to process commands for %d user(s) across %d server(s)\n", len(event.Users), len(event.Servers))
		fmt.Print(readyLogMessage)
		bot.logger.Log(readyLogMessage)
	})

	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventMessage) {
		// the bot should never react to its own messages
		if event.Author == s.State.Self().ID {
			return
		}

		// build message context
		ctx, err := NewContext(s, event)
		if err != nil {
			fmt.Println("Error building context: ", err)
			bot.logger.Log("error building context: " + err.Error())
			return
		}

		// hand context off to handler
		bot.handlerEventMessage(ctx)
	})

	err = session.Open()
	if err != nil {
		panic(err)
	}

	// let the bot run by awaiting signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
