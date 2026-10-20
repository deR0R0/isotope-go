package commands

import (
	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func init() {
	RegisterCommand(
		discord.SlashCommandCreate{
			Name:        "logout",
			Description: "Unlinks your Ion account from your discord account",
		},
		"/logout",
		logoutCommandHelper,
	)
}

func handleLogout(user discord.User) error {
	err := db.ClearTokenData(db.GetDB(), user.ID.String())
	return err
}

func logout(event *handler.CommandEvent) error {
	// check if they're logged in
	result, err := login(event.User().ID.String())

	if err != nil {
		event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("Internal Error.").WithEphemeral(true),
		)
		return err
	}

	if !result.LoggedIn {
		event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("You're not logged in.").WithEphemeral(true),
		)
		return nil
	}

	err = handleLogout(event.User())
	if err != nil {
		event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("Internal Error.").WithEphemeral(true),
		)
		return err
	}

	_, err = event.Client().Rest.CreateFollowupMessage(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageCreate().WithContent("You have successfully logged out.").WithEphemeral(true),
	)

	return err
}

func logoutCommandHelper(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	event.DeferCreateMessage(true)
	return logout(event)
}
