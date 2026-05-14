package commands

import (
	"log/slog"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
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

// local error message to make it quick!
func loginShowErrorMessage(event *events.ApplicationCommandInteractionCreate) {
	ShowErrorMessage("login", func() error {
		_, err := event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("Internal Server Error"),
		)
		return err
	})
}

func handleLogin(event *events.ApplicationCommandInteractionCreate) {
	// first, defer the interaction. show the "<bot> is thinking..." message (this may be a long-running task)
	event.DeferCreateMessage(true)

	userID := event.User().ID.String()

	// check if user exists in the db and verify that they actually have data in access token
	exists, err := db.UserExists(db.GetDB(), userID)
	if exists {
		at, _, _, _, err := db.GetTokenData(db.GetDB(), userID)

		if err != nil {
			loginShowErrorMessage(event)
			return
		}

		if at != "" {
			event.Client().Rest.CreateFollowupMessage(
				event.ApplicationID(),
				event.Token(),
				discord.NewMessageCreate().WithContent("You're already logged in!"),
			)
			return
		}
	}

	session, err := oauth.Manager().CreateNewSession(event.User().ID.String())

	if err != nil {
		loginShowErrorMessage(event)
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

	if !exists {
		db.CreateUser(db.GetDB(), userID)
	}

	db.SetState(db.GetDB(), userID, session.State)
}
