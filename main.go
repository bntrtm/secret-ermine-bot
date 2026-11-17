package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bntrtm/secret-ermine-bot/internal/logging"
	"github.com/bntrtm/secret-ermine-bot/model"
	"github.com/bntrtm/secret-ermine-bot/repo"
	"github.com/joho/godotenv"

	// 'sgo' as in "stoat go"
	sgo "github.com/sentinelb51/revoltgo"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed sql/schema/*.sql
var embedMigrations embed.FS

func connectToDB(migrationsDir string) (*sql.DB, error) {
	dbPath := "data/"
	if err := os.MkdirAll(dbPath, 0o755); err != nil {
		return nil, err
	}
	sebDB, err := sql.Open("sqlite3", dbPath+"seb.db")
	if err != nil {
		slog.Error("failed to open database", slog.Any("err", err))
		return nil, err
	}

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("failed to set goose dialect", slog.Any("err", err))
		return nil, err
	}
	if err := goose.Up(sebDB, migrationsDir); err != nil {
		slog.Error("failed to run migrations", slog.Any("err", err))
		return nil, err
	}

	return sebDB, nil
}

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
		slog.Warn("could not load .env file", slog.Any("err", err))
	}

	sebDB, err := connectToDB("sql/schema")
	if err != nil {
		os.Exit(1)
	}
	defer sebDB.Close()

	bot := &botStore{
		Events:              map[string]model.SecretSantaEvent{},
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
	if bot.repo, err = repo.New(sebDB); err != nil {
		panic(err)
	}

	// start a new sgo session
	session := sgo.New(bot.Token)

	// structured logging setup
	bot.logger = &logging.Logger{}
	bot.logger.Init()
	defer bot.logger.Quit()

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

	bot.loadEvents()

	// let the bot run by awaiting signals
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
