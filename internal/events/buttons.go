package events

import (
	"github.com/deR0R0/isotope-go/internal/commands"
	"github.com/disgoorg/disgo/events"
)

func ButtonListener(event *events.ComponentInteractionCreate) {
	if event.ButtonInteractionData().CustomID() == "isotope_authorize" {
		event.DeferCreateMessage(true)
		commands.HandleLogin(&commands.LoginParams{UserID: event.User().ID.String(), Rest: event.Client().Rest, ApplicationID: event.ApplicationID(), Token: event.Token()})
	}
}