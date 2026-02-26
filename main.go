package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"
)

// botStore tracks persistent data related to the bot's activity across one or more servers
type botStore struct {
	session *sgo.Session

	Token  string
	Events map[string]SecretSantaSession // map of servers to secret-santa events (limited to one active SSE/server)
}

func main() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	bot := &botStore{
		Events: make(map[string]SecretSantaSession),
		Token:  os.Getenv("BOT_TOKEN"),
	}

	// start a new sgo session
	bot.session = sgo.New(bot.Token)

	sgo.AddHandler(bot.session, func(s *sgo.Session, event *sgo.EventReady) {
		fmt.Printf("Ready to process commands for %d user(s) across %d server(s)\n", len(event.Users), len(event.Servers))
	})
	sgo.AddHandler(bot.session, func(s *sgo.Session, event *sgo.EventMessage) {
		handleHelpMessage(s, event)
	})
	sgo.AddHandler(bot.session, func(s *sgo.Session, event *sgo.EventMessage) {
		handlePingMessage(s, event)
	})

	sgo.AddHandler(bot.session, bot.handlerNewSantaSession)

	err = bot.session.Open()
	if err != nil {
		panic(err)
	}

	// let the bot run by awaiting signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

type Participant struct {
	Username    string `json:"username"`     // Stoat username of this participant in the server
	SecretSanta string `json:"secret_santa"` // user tasked with getting this participant a gift
	Giftee      string `json:"giftee"`       // user this participant is tasked with giving a gift to
	About       string `json:"about"`        // a short message from this participant addressing gift ideas for them
}

type SecretSantaSession struct {
	Organizer        *sgo.User     `json:"organizer"`         // user that started the session
	OrganizationDate string        `json:"organization_date"` // timestring referring to the date & time the session began
	DistributionDate string        `json:"distribution_date"` // timestring referring to the date & time gifts will be distributed
	Participants     []Participant `json:"participants"`      // list of users participating in the Session

	// Spend limit not enforced with Int parsing and such because currency validation
	// for something this simple just seems unnecessary
	SpendLimit string // a user-input string detailing monetary spending limits for the Secret Santa event
}

// handlerNewSantaSession tells the bot it's time for a new Secret Santa Session!
// usage: !new <date (YYYY-MM-DD)> <spend_limit>

func (b *botStore) handlerNewSantaSession(session *sgo.Session, msg *sgo.EventMessage) {
	var content string
	var err error

	// always try to send ANY existing message in the content buffer, if present
	defer func() {
		if content == "" {
			return
		}
		send := sgo.MessageSend{
			Content: content,
		}

		message, err := session.ChannelMessageSend(msg.Channel, send)
		if err != nil {
			fmt.Println("Error sending message: ", err)
		}

		fmt.Println("Sent message: ", message.Content)
	}()

	if !strings.HasPrefix(msg.Content, "!new") {
		return
	}

	fields := strings.Split(msg.Content, " ")
	if fields[0] != "!new" {
		content = fmt.Sprintf("Unknown command '%s', use '!help' for all available commands.", fields[0])
		return
	}

	argCount := len(fields) - 1

	if argCount != 2 {
		content = fmt.Sprintf("Missing arguments; expected 2, but got %d", argCount)
		return
	}

	distributionDate, err := time.Parse("2006-01-02", fields[1])
	if err != nil {
		fmt.Println("Could not parse distribution date provided as time.Time")
		content = fmt.Sprintf("Bad date input '%s'. Please use the format: YYYY-MM-DD", fields[1])
		return
	}

	newSSE := &SecretSantaSession{}
	newSSE.OrganizationDate = time.Now().String()
	newSSE.Organizer, err = session.User(msg.Author)
	if err != nil {
		newSSE.Organizer = &sgo.User{Username: "UNKNOWN"}
	}
	newSSE.DistributionDate = distributionDate.Format("2006-01-02")

	content = fmt.Sprintf("%s is organizing a Secret Santa event! It will take place on %s.", newSSE.Organizer.Mention(), newSSE.DistributionDate)
	content += "\nTo join, react to this message!"
	content += fmt.Sprintf("\nPlease limit your spending according to the following: %s", newSSE.SpendLimit)
	content += fmt.Sprintf("\n%s, you can start the event whenever you're ready with '!start', so long as at least THREE participants have joined.", newSSE.Organizer.Mention())
	content += "\nOr, the event can be canceled with '!cancel'."
}

func handleHelpMessage(session *sgo.Session, msg *sgo.EventMessage) {
	if msg.Content != "!help" {
		return
	}

	content := "Available commands: !help !ping"

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
