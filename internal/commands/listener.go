package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/events"
)

func Listener(event *events.ApplicationCommandInteractionCreate) {
	slog.Info("executed command", "user", event.User().Username, "command", event.SlashCommandInteractionData().CommandName())
	switch(event.SlashCommandInteractionData().CommandName()) {
	case "ping":
		handlePing(event)
	case "login":
		handleLogin(event)
	}
}