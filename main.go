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

	botToken := os.Getenv("BOT_TOKEN")

	// start a new session
	session := sgo.New(botToken)

	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventReady) {
		fmt.Printf("Ready to process commands for %d user(s) across %d server(s)\n", len(event.Users), len(event.Servers))
	})

	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventMessage) {
		handleHelpMessage(s, event)
	})
	sgo.AddHandler(session, func(s *sgo.Session, event *sgo.EventMessage) {
		handlePingMessage(s, event)
	})

	err = session.Open()
	if err != nil {
		panic(err)
	}

	// let the bot run by awaiting signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func handleHelpMessage(session *sgo.Session, msg *sgo.EventMessage) {
	if msg.Content != "!help" {
		return
	}

	latency := session.WS.Latency()
	content := "Available commands: !help !ping"

	if latency.Milliseconds() == 0 {
		content = "Still calculating, keep re-trying this command in 15-second intervals."
	}

	send := sgo.MessageSend{
		Content: content,
	}

	message, err := session.ChannelMessageSend(msg.Channel, send)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return
	}

	fmt.Println("Sent message: ", message.Content)
}

func handlePingMessage(session *sgo.Session, msg *sgo.EventMessage) {
	if msg.Content != "!ping" {
		return
	}

	latency := session.WS.Latency()
	content := latency.String()

	if latency.Milliseconds() == 0 {
		content = "Still calculating, keep re-trying this command in 15-second intervals."
	}

	send := sgo.MessageSend{
		Content: content,
	}

	message, err := session.ChannelMessageSend(msg.Channel, send)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		return
	}

	fmt.Println("Sent message: ", message.Content)
}
