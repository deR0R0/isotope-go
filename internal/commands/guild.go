package commands

import (
	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
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

type guildSettingsSelectResult struct {
	Select *discord.StringSelectMenuComponent
	VerifySystem bool
	Role *discord.Role
}

func getGuildSettingsMessage(result *guildSettingsSelectResult, guildName *string) (string) {
	var message MessageBuilder

	message.AddLargeHeader(*guildName + " Settings")
	message.AddSeperators()

	if result.VerifySystem {
		message.AddMediumHeader(":white_check_mark: Verify System Enabled")
		if result.Role != nil { message.AddIndent("Role: " + result.Role.Mention()) } else { message.AddIndent("Role: Not Set") }
	} else {
		message.AddMediumHeader(":x: Verify System Disabled")
	}

	return message.BuildMessage()
}

func getGuildSettingsSelect(guild snowflake.ID) (*guildSettingsSelectResult, error) {
	result := &guildSettingsSelectResult{}
	selectMenuOptions := []SelectOptions{}

	// get the data from the db
	verifyEnabled, err := db.GetBoolFromGuilds(db.GetDB(), guild.String(), "verify_enabled")
	if err != nil {
		return nil, err
	}

	result.VerifySystem = verifyEnabled

	// verify system stuff
	if verifyEnabled {
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Disable Verify System", Emoji: &discord.ComponentEmoji{Name: "🔴"}})
		// role
		role, err := GetVerifyRoleFromGuild(&guild)

		if err != nil {
			// TODO: clear db of the role snowflake
			return nil, err
		}

		result.Role = role

		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Verify Role"})
	} else {
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Enable Verify System", Emoji: &discord.ComponentEmoji{Name: "🟢"}})
	}

	selectMenu := CreateNewSelect("guild_settings", "Select A Setting...", guildSettingsSelectMenu, selectMenuOptions...)

	result.Select = selectMenu

	return result, nil
}

func guildSettings(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	guild, ok := event.Guild()
	if !ok {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be run in a guild/server!"))
	}

	result, err := getGuildSettingsSelect(*event.GuildID())
	if err != nil {
		event.CreateMessage(discord.NewMessageCreate().WithContent("Internal Error :("))
		return err
	}

	message := getGuildSettingsMessage(result, &guild.Name)

	return event.CreateMessage(
		discord.NewMessageCreate().
			WithContent(message).
			WithComponents(
				discord.NewActionRow(
					result.Select,
				),
			).
			WithAllowedMentions(&discord.AllowedMentions{
				Parse: []discord.AllowedMentionType{},
			}), // NEED THIS SO IT DOESN'T PING THE ROLE
	)
}

func guildSettingsSelectMenu(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	guild, ok := event.Guild()
	if !ok {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be run in a guild/server!"))
	}

	event.DeferUpdateMessage() // discord requires an acknowledgement before we can edit the original message

	// determine what to do
	menuData := data.(discord.StringSelectMenuInteractionData)
	values := menuData.Values

	selectMenu, err := getGuildSettingsSelect(*event.GuildID()) // "previous" select TODO: REWORK BY SAVING THIS SELECT

	// TEMP WAY TO HANDLE THIS. NEXT TIME REWORK HOW SELECT MENUS WORK INTERNALLY
	switch values[0] {
	case "option_0": // verify system
		if selectMenu.VerifySystem {
			err := db.SetToGuild(db.GetDB(), event.GuildID().String(), "verify_enabled", false)
			if err != nil {
				return err
			}
		} else {
			err := db.SetToGuild(db.GetDB(), event.GuildID().String(), "verify_enabled", true)
			if err != nil {
				return err
			}
		}
	}

	// update the select
	selectMenu, err = getGuildSettingsSelect(*event.GuildID())

	if err != nil {
		event.UpdateInteractionResponse(
			discord.NewMessageUpdate().
			WithContent("Internal Error.").
			WithComponents(),
		)
	}

	message := getGuildSettingsMessage(selectMenu, &guild.Name)

	event.UpdateInteractionResponse(
			discord.NewMessageUpdate().
			WithContent(message).
			WithComponents(
				discord.NewActionRow(
					selectMenu.Select,
				),
			),
		)
	
	return err
}
