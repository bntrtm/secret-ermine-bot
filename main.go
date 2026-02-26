package main

import (
	"fmt"
	"log"
	"math/rand"
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
		Events: map[string]SecretSantaEvent{},
		Token:  os.Getenv("BOT_TOKEN"),
	}

	// start a new sgo session
	bot.session = sgo.New(bot.Token)

	sgo.AddHandler(bot.session, func(s *sgo.Session, event *sgo.EventReady) {
		fmt.Printf("Ready to process commands for %d user(s) across %d server(s)\n", len(event.Users), len(event.Servers))
	})

	sgo.AddHandler(bot.session, bot.handlerEventMessage)

	err = bot.session.Open()
	if err != nil {
		panic(err)
	}

	// let the bot run by awaiting signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func testGifteeAssignment() {
	recorded := []string{"user1", "user2", "user3"}
	// shuffle recorded set of participants
	for i := range recorded {
		j := rand.Intn(i + 1)
		recorded[i], recorded[j] = recorded[j], recorded[i]
	}
	fmt.Println(recorded)

	type part struct {
		Giftee      string
		SecretSanta string
	}

	parts := map[string]part{
		"user1": {},
		"user2": {},
		"user3": {},
	}

	for i, uID := range recorded {
		if pt, ok := parts[uID]; ok {
			gifteeIndex := i + 1
			// if I'm last, my giftee is the first participant
			if i == len(recorded)-1 {
				gifteeIndex = 0
			}
			pt.Giftee = recorded[gifteeIndex]
			// assuming they're a participant...
			if ptGiftee, ok := parts[pt.Giftee]; ok {
				// tell the system I'm their Secret Santa...
				ptGiftee.SecretSanta = uID
				parts[pt.Giftee] = ptGiftee
			}
			// ...and tell the system I know who my giftee is
			parts[uID] = pt
		}
	}

	debugLine := "SSE RESULTS: "
	for uID, p := range parts {
		debugLine += fmt.Sprintf("\n%s is the Secret Santa of giftee: %s. Their Secret Santa is: %s", uID, p.Giftee, p.SecretSanta)
	}
	fmt.Println(debugLine)

	os.Exit(0)
}
