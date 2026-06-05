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
	event.CreateMessage(discord.NewMessageCreate().WithContent("guild settings command"))
	return nil
}