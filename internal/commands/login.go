package commands

import (
	"log/slog"
	"time"

	"github.com/deR0R0/isotope-go/internal/oauth"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	Register(discord.SlashCommandCreate{
		Name:        "login",
		Description: "Log into your Ion account from this bot.",
	})
}

func handleLogin(event *events.ApplicationCommandInteractionCreate) {
	// first, defer the interaction. show the "<bot> is thinking..." message (this may be a long-running task)
	event.DeferCreateMessage(true)

	session, err := oauth.Manager().CreateNewSession(event.User().ID.String())

	if err != nil {
		slog.Info("sending err message to user and stopping rest of command logic...")
		event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("Internal Server Error"),
		)
		return
	}

	msg, err := event.Client().Rest.CreateFollowupMessage(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageCreate().WithContent(session.RedirectURI),
	)

	if err != nil {
		slog.Info("error while sending response: ", slog.Any("err", err))
		return
	}

	DeleteAfter(5*time.Second, func() error {
		return event.Client().Rest.DeleteFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			msg.ID,
		)
	})
}
