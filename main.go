package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/deR0R0/isotope-go/internal/commands"
	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/deR0R0/isotope-go/internal/oauth"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.Init()
	oauth.Init()

	var DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")

	// create the client
	slog.Info("creating the client")
	client, err := disgo.New(DISCORD_TOKEN,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(commands.Listener),
	)

	if err != nil {
		slog.Error("error while making disgo instance: ", slog.Any("err", err))
		return
	}

	defer client.Close(context.TODO())

	// command loading
	if os.Getenv("APP_ENV") == "development" { // if we're in a development environment, only sync our commands with a certain guild
		slog.Info("syncing commands to TEST GUILD in a DEVELOPMENT environment.")
		guildId := snowflake.GetEnv("GUILD_TEST_ID")
		if guildId == 0 {
			slog.Error("missing environmental var \"GUILD_TEST_ID\" in a development environment.")
		} else {
			client.Rest.SetGuildCommands(client.ApplicationID, guildId, commands.Commands)
		}
	} else {
		slog.Info("Syncing commands to GLOBAL in a DEVELOPMENT environment.")
		client.Rest.SetGlobalCommands(client.ApplicationID, commands.Commands)
		commands.Commands = nil // garbage collect this as we don't need this anymore
	}

	// actually connect to gateway
	slog.Info("connecting to gateway...")
	err = client.OpenGateway(context.TODO())
	if err != nil {
		slog.Error("Could not connect to gateway.", slog.Any("err", err))
		panic("Could not connect to gateway.")
	}

	if selfUser, ok := client.Caches.SelfUser(); ok {
		slog.Info("Logged in", slog.Any("user", selfUser))
	}

	// dont exit. wait for quit via ctrl c
	slog.Info("bot successfully connected. Waiting for exit via CTRL + C")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-s
}
