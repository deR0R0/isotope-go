package commands

// TODO: make an easy way to make a setting. (createGuildSetting(something, soemthing, something))

import (
	"log/slog"

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
	Select       *discord.StringSelectMenuComponent
	VerifySystem bool
	Role         *discord.Role
	Channel *discord.Channel
}

func getGuildSettingsMessage(result *guildSettingsSelectResult, guildName *string, opts ...string) string {
	var message MessageBuilder

	// "opts" will be what messages that will go before this.
	for _, opt := range opts {
		message.AddCodeBlock(opt)
	}

	message.AddMediumHeader(*guildName + " Settings")

	if result.VerifySystem {
		message.AddMessage("**Verify System**: Enabled :white_check_mark:")
		if result.Role != nil {
			message.AddIndent("**Role**: " + result.Role.Mention())
		} else {
			message.AddIndent("**Role**: Not Set")
		}

		if result.Channel != nil {
			message.AddIndent("**Channel**: <#" + (*result.Channel).ID().String() + ">")
		} else {
			message.AddIndent("**Channel**: Not Set")
		}
	} else {
		message.AddMessage("**Verify System**: Disabled :x:")
	}

	return message.BuildMessage()
}

// populates a select menu with options
func getGuildSettingsSelect(guild snowflake.ID, author snowflake.ID) (*guildSettingsSelectResult, error) {
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
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Disable Verify System", Value: "Verify", Emoji: &discord.ComponentEmoji{Name: "🔴"}})
		// role
		role, err := GetVerifyRoleFromGuild(&guild)

		if err != nil {
			// TODO: clear db of the role snowflake
			return nil, err
		}

		result.Role = role

		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Verify Role", Value: "VerifyRole", Description: "The role given to members who verified thru ion", Emoji: &discord.ComponentEmoji{Name: "✍️"}})

		// channel
		channel, err := GetVerifyChannelFromGuild(&guild)

		if err != nil {
			return nil, err // TODO: same as role, clear when error out otherwise always error.
		}

		result.Channel = channel

		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Verify Channel", Value: "VerifyChannel", Description: "The channel where the verify button should be sent", Emoji: &discord.ComponentEmoji{Name: "✍️"}})
	} else {
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Enable Verify System", Value: "Verify", Description: "Allows you to have Ion verification", Emoji: &discord.ComponentEmoji{Name: "🟢"}})
	}

	selectMenu := CreateRestrictedSelect(60, author, "guild_settings", "Select A Setting...", guildSettingsSelectMenu, selectMenuOptions...)

	result.Select = selectMenu

	return result, nil
}

func updateGuildSettingsMessage(event *handler.ComponentEvent, guild discord.Guild) error {
	event.DeferUpdateMessage() // discord requires an acknowledgement before we can edit the original message

	// update the select
	selectMenu, err := getGuildSettingsSelect(*event.GuildID(), event.User().ID)

	if err != nil {
		event.UpdateInteractionResponse(
			discord.NewMessageUpdate().
				WithContent("Internal Error.").
				WithComponents(),
		)
	}

	// get the mesage dpedning on the select menu results
	message := getGuildSettingsMessage(selectMenu, &guild.Name)

	event.UpdateInteractionResponse(
		discord.NewMessageUpdate().
			WithContent(message).
			WithComponents(
				discord.NewActionRow(
					selectMenu.Select,
				),
			).WithAllowedMentions(&discord.AllowedMentions{
			Parse: []discord.AllowedMentionType{},
		}),
	)
	
	return err
}

func guildSettings(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	guild, ok := event.Guild()
	if !ok {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be run in a guild/server!"))
	}

	result, err := getGuildSettingsSelect(*event.GuildID(), event.User().ID)
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

func doActionBasedOnSelect(option string, selectMenu *guildSettingsSelectResult, guild discord.Guild, event *handler.ComponentEvent) error {
	var err error

	switch option {
	case "Verify": // verify system
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

		updateGuildSettingsMessage(event, guild)
	case "VerifyRole":
		event.DeferUpdateMessage()
		// register a "role select" menu and send it
		roleSelectMenu := CreateRoleSelect(60, event.User().ID, "guild_settings_role", "Select a role for verified members", handleGuildSettingsRoleSelect)

		message := getGuildSettingsMessage(selectMenu, &guild.Name, "Choose the role that would be given to verified members in the dropdown... (doesn't include the channel type you want? contact @robboach)")

		_, err = event.UpdateInteractionResponse(
			discord.NewMessageUpdate().
				WithContent(message).
				WithComponents(
					discord.NewActionRow(
						roleSelectMenu,
					),
					discord.NewActionRow(
						CreateNewRestrictedButton(60, event.User().ID, "cancel", "Cancel", discord.ButtonStyleDanger, handleGuildSettingsCancel),
					),
				).WithAllowedMentions(&discord.AllowedMentions{
				Parse: []discord.AllowedMentionType{},
			}),
		)
	case "VerifyChannel":
		event.DeferUpdateMessage()

		channelSelectMenu := CreateChannelSelect(60, event.User().ID, "guild_settings_channel", "Select a channel for the button", handleGuildSettingsChannelSelect)

		message := getGuildSettingsMessage(selectMenu, &guild.Name, "Choose the channel that the verify button will be sent in...")

		_, err = event.UpdateInteractionResponse(
			discord.NewMessageUpdate().
				WithContent(message).
				WithComponents(
					discord.NewActionRow(
						channelSelectMenu,
					),
					discord.NewActionRow(
						CreateNewRestrictedButton(60, event.User().ID, "cancel", "Cancel", discord.ButtonStyleDanger, handleGuildSettingsCancel),
					),
				).WithAllowedMentions(&discord.AllowedMentions{
				Parse: []discord.AllowedMentionType{},
			}),
		)
	}

	return err
}

func guildSettingsSelectMenu(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	// decode the id params
	selectMenuID := data.CustomID()
	selectMenuArgs := DecodeRouteArgs(selectMenuID)

	// author check
	if selectMenuArgs["author"] != event.User().ID.String() {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This is not your select menu!").WithEphemeral(true))
	}

	// guild check
	guild, ok := event.Guild()
	if !ok {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be run in a guild/server!").WithEphemeral(true))
	}

	// determine what to do
	menuData := data.(discord.StringSelectMenuInteractionData)
	values := menuData.Values

	selectMenu, err := getGuildSettingsSelect(*event.GuildID(), event.User().ID) // "previous" select TODO: REWORK BY SAVING THIS SELECT

	// handle different values based on their custom values
	doActionBasedOnSelect(values[0], selectMenu, guild, event)

	return err
}

func handleGuildSettingsRoleSelect(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	// cast select role menu
	roleData, ok := data.(discord.RoleSelectMenuInteractionData)

	if !ok {
		slog.Error("call to guild settings role select handler was made without a select menu context/whatever")
		return event.CreateMessage(discord.NewMessageCreate().WithContent("Internal error.").WithEphemeral(true))
	}

	// check for a guild
	guild, ok := event.Guild()
	if !ok {
		slog.Info("call to guild settings role select was not to a guild.")
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be used in a guild.").WithEphemeral(true))
	}

	// assume they only selected 1
	roleID := roleData.Values[0]

	// add to the db
	db.SetToGuild(db.GetDB(), guild.ID.String(), "verify_role_id", roleID.String())

	return updateGuildSettingsMessage(event, guild)
}

func handleGuildSettingsChannelSelect(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	// cast select role menu
	channelData, ok := data.(discord.ChannelSelectMenuInteractionData)

	if !ok {
		slog.Error("call to guild settings channel select handler was made without a select menu context/whatever")
		return event.CreateMessage(discord.NewMessageCreate().WithContent("Internal error.").WithEphemeral(true))
	}

	// check for a guild
	guild, ok := event.Guild()
	if !ok {
		slog.Info("call to guild settings role select was not to a guild.")
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be used in a guild.").WithEphemeral(true))
	}

	// assume they only selected 1
	channelID := channelData.Values[0]

	// add to the db
	db.SetToGuild(db.GetDB(), guild.ID.String(), "channel_id", channelID.String())

	return updateGuildSettingsMessage(event, guild)
}

func handleGuildSettingsCancel(data discord.ButtonInteractionData, event *handler.ComponentEvent) error {
	guild, ok := event.Guild()
	if !ok {
		return event.CreateMessage(discord.NewMessageCreate().WithContent("This command can only be run in a guild/server!").WithEphemeral(true))
	}

	return updateGuildSettingsMessage(event, guild)
}
