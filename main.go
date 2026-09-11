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

// getBaseURL returns the value set by the BASE_URL
// environment variable, or falls back to the public
// `stoat.chat` base URL.
func getBaseURL() string {
	envVal := os.Getenv("BASE_URL")
	if envVal == "" {
		return "https://stoat.chat/"
	} else {
		return fmt.Sprintf("%s/", strings.TrimRight(envVal, "/"))
	}
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
		commands: map[string]command{},
		baseUrl:  getBaseURL(),
		platform: os.Getenv("PLATFORM"),
	}
	bot.initCommands()

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
