package events

import (
	"log/slog"

	"github.com/disgoorg/disgo/events"
	"github.com/deR0R0/isotope-go/internal/commands"
)

func on_ready(event *events.Ready) {
	slog.Info("logged in as user " + event.User.Username)
	slog.Info("bot is ready to recieve commands")
	// register our permannet button
	commands.RegisterButton("/button/isotope_authorize", commands.LoginButtonHandler)	
}