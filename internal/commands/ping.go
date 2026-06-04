package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func init() {
	RegisterCommand(
		discord.SlashCommandCreate{
			Name: "ping",
			Description: "Sends the roundtrip ping from this bot to Discord servers.",
		},
		"/ping",
		handlePing,
	)
}

func handlePing(data discord.SlashCommandInteractionData, event *handler.CommandEvent) (error) {
	err := event.CreateMessage(discord.NewMessageCreate().WithContent("Pong :ping_pong:!\nLatency: " + event.Client().Gateway.Latency().String()))
	if err != nil {
		slog.Error("couldn't send response", slog.Any("err", err))
		return err
	}
	return nil
}
