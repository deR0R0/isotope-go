package commands

import (
	"log/slog"

	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func init() {
	RegisterCommand(
		discord.SlashCommandCreate{
			Name:        "guild",
			Description: "Guild command group",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name:        "settings",
					Description: "Show the guild settings",
				},
			},
		},
		"/guild/settings", guildSettings,
	)
}

func guildSettings(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	// get the data from the db
	enabled, err := db.GetBoolFromGuilds(db.GetDB(), event.GuildID().String(), "verify_enabled")
	if err != nil {
		event.CreateMessage(discord.NewMessageCreate().WithContent("Internal Error!"))
		return err
	}

	if enabled{
		slog.Info("enabled")
	}


	selectMenu := CreateNewSelect("guild_settings", "Select A Setting...", guildSettingsSelectMenu, ":green_circle: option 1", ":red_circle: option 2")

	event.CreateMessage(
		discord.NewMessageCreate().
			WithContent("guild settings command").
			WithComponents(
				discord.NewActionRow(
					selectMenu,
				),
			),
	)
	return nil
}

func guildSettingsSelectMenu(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	event.DeferUpdateMessage() // discord requires an acknowledgement before we can edit the original message

	// build menu stuff and send
	menuData := data.(discord.StringSelectMenuInteractionData)
	values := menuData.Values
	event.UpdateInteractionResponse(discord.NewMessageUpdate().WithContent("selected: `" + values[0] + "`").WithComponents())
	
	return nil
}
