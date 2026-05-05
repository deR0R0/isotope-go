package commands

import (
	"log/slog"

	"github.com/deR0R0/isotope-go/internal/oauth"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	Register(discord.SlashCommandCreate{
		Name: "login",
		Description: "Generates a link to login to Ion.",
	})
}

func handleLogin(event *events.ApplicationCommandInteractionCreate) {
	// first, defer the interaction. show the "<bot> is thinking..." message (this may be a long-running task)
	event.DeferCreateMessage(true)

	session, err := oauth.Manager().CreateNewSession(event.User().ID.String())

	slog.Info("redirect url", slog.String("url", session.RedirectURI))

	_, err = event.Client().Rest.CreateFollowupMessage(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageCreate().WithContent(session.RedirectURI),
	)

	if err != nil {
		slog.Info("error while sending response: ", slog.Any("err", err))
	}
}