package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

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
	}

	// start a new sgo session
	session := sgo.New(bot.Token)

	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventReady) {
		fmt.Printf("Ready to process commands for %d user(s) across %d server(s)\n", len(event.Users), len(event.Servers))
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
