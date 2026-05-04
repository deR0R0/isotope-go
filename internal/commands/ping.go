package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	Register(discord.SlashCommandCreate{
		Name: "ping",
		Description: "Returns the gateway latency of this bot.",
	})
}

func handlePing(event *events.ApplicationCommandInteractionCreate) {
	err := event.CreateMessage(discord.NewMessageCreate().WithContent("Pong :ping_pong:!\nLatency: " + event.Client().Gateway.Latency().String()))
	if err != nil {
		slog.Error("couldn't send response", slog.Any("err", err))
	}
}