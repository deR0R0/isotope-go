package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func init() {
	RegisterCommand(
		discord.SlashCommandCreate{
			Name: "guild",
			Description: "Guild command group",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name: "settings",
					Description: "Show the guild settings",
				},
			},
		},
		"/guild/settings", guildSettings,
	)
}

func guildSettings(data discord.SlashCommandInteractionData, event *handler.CommandEvent) (error) {
	selectMenu := CreateNewSelect("guild_settings", "Select A Setting...", guildSettingsSelectMenu, "option 1", "option 2")

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
	menuData := data.(discord.StringSelectMenuInteractionData)
	values := menuData.Values
	event.CreateMessage(discord.NewMessageCreate().WithContent("selected: `" + values[0] + "`"))
	return nil
}