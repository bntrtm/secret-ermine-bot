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
func (b *botStore) handleNewSantaEventMessage(args []string, msg *sgo.EventMessage) string {
	var content string

	if len(args) != 2 {
		content = fmt.Sprintf("Argument mismatch; expected 2, but got %d", len(args))
		return content
	}

	dateInput := args[0]
	spendLimit := args[1]

	distributionDate, err := time.Parse("2006-01-02", dateInput)
	if err != nil {
		fmt.Println("Could not parse distribution date provided as time.Time")
		content = fmt.Sprintf("Bad date input '%s'. Please use the format: YYYY-MM-DD", dateInput)
		return content
	}

	newSSE := &SecretSantaSession{}
	newSSE.OrganizationDate = time.Now().String()
	newSSE.Organizer, err = b.session.User(msg.Author)
	if err != nil {
		newSSE.Organizer = &sgo.User{Username: "UNKNOWN"}
	}
	newSSE.DistributionDate = distributionDate.Format("2006-01-02")
	newSSE.SpendLimit = spendLimit

	content = fmt.Sprintf("%s is organizing a Secret Santa event! It will take place on %s.", newSSE.Organizer.Mention(), newSSE.DistributionDate)
	content += "\nTo join, react to this message!"
	content += fmt.Sprintf("\nPlease limit your spending according to: %s", newSSE.SpendLimit)
	content += fmt.Sprintf("\n%s, you can start the event whenever you're ready with '!start', so long as at least THREE participants have joined.", newSSE.Organizer.Mention())
	content += "\nOr, the event can be canceled with '!cancel'."

	return content
}

func (b *botStore) handlerEventMessage(session *sgo.Session, msg *sgo.EventMessage) {
	var content string

	// always try to send ANY existing message in the content buffer, if present
	defer func() {
		if content == "" {
			return
		}
		send := sgo.MessageSend{
			Content: content,
		}

		message, err := b.session.ChannelMessageSend(msg.Channel, send)
		if err != nil {
			fmt.Println("Error sending message: ", err)
		}

		fmt.Println("Sent message: ", message.Content)
	}()

	if !strings.HasPrefix(msg.Content, "!") {
		return
	}

	fields := strings.Split(msg.Content, " ")
	command, args := strings.TrimPrefix(fields[0], "!"), fields[1:]
	switch command {
	case "new":
		content = b.handleNewSantaEventMessage(args, msg)
	case "help":
		content = b.handleMsgHelp()
	case "ping":
		content = b.handleMsgPing()
	default:
		content = fmt.Sprintf("Unknown command '%s', use '!help' for all available commands.", fields[0])
	}
}

func (b *botStore) handleMsgHelp() string {
	return "Available commands: !help !ping !new"
}

func (b *botStore) handleMsgPing() string {
	latency := b.session.WS.Latency()

	if latency.Milliseconds() == 0 {
		return "Still calculating, keep re-trying this command in 15-second intervals."
	}

	return b.session.WS.Latency().String()
}
