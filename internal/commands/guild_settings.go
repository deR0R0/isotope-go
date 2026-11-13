package commands

// TODO: make an easy way to make a setting. (createGuildSetting(something, soemthing, something))

import (
	//"log/slog"

	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
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
		"/guild/settings", handleGuildSettingsCommand,
	)
}

type guildSettings struct {
	Select        *discord.StringSelectMenuComponent
	LogChannel    *discord.GuildTextChannel
	VerifySystem  *bool
	VerifyRole    *discord.Role
	VerifyChannel *discord.GuildTextChannel
}

/*
Handlers (command/button) to route to the proper internal method

Difference between command/button (or select menu) is that the command will create the initial message
which would use the defer create message while buttons and select menus would UPDATE the message, so
we defer the update message instead.
*/

func handleGuildSettingsCommand(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	event.DeferCreateMessage(false)
	// get guild
	guild, ok := event.Guild()
	if !ok {
		event.UpdateInteractionResponse(discord.NewMessageUpdate().WithContent("This command can only be run in a guild."))
		return nil
	}

	if !event.Member().Permissions.Has(discord.PermissionManageGuild) {
		event.UpdateInteractionResponse(discord.NewMessageUpdate().WithContent("You don't have permission to use this command."))
		return nil
	}

	// get the settings + select
	return sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User)
}

// TODO: holy fuck this is an ugly function. i'll fix later.
func handleGuildSettingsSelect(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error {
	event.DeferUpdateMessage()

	// check guild - this shouldn't ever fail but this is here :)
	guild, ok := event.Guild()
	if !ok {
		event.UpdateInteractionResponse(discord.NewMessageUpdate().WithContent("This command can only be run in a guild.").WithComponents())
		return nil
	}

	// check expiry and author and permissions
	selectRoute := data.CustomID()
	properties := getSelectFromLookup(selectRoute)
	if properties == nil {
		event.UpdateInteractionResponse(discord.NewMessageUpdate().WithContent("Sorry, this interaction is expired.").WithComponents())
		return nil
	}

	if !slices.Contains(properties.Users, event.User().ID.String()) {
		event.CreateFollowupMessage(discord.NewMessageCreate().WithContent("You can't interact with this select menu.").WithEphemeral(true))
		return nil
	}

	if !event.Member().Permissions.Has(discord.PermissionManageGuild) {
		event.CreateFollowupMessage(discord.NewMessageCreate().WithContent("You don't have permission to use this command.").WithEphemeral(true))
		return nil
	}

	var selectedString *string
	var selectedRole *snowflake.ID
	var selectedChannel *snowflake.ID

	// switch the select menu interaction data
	switch data.(type) {
	case discord.StringSelectMenuInteractionData:
		casted, ok := data.(discord.StringSelectMenuInteractionData)
		if !ok {
			slog.Error("didn't expect a specific type (discord.StringSelectMenuInteractionData)")
			return fmt.Errorf("cast error")
		}
		selectedString = &casted.Values[0]
	case discord.RoleSelectMenuInteractionData:
		casted, ok := data.(discord.RoleSelectMenuInteractionData)
		if !ok {
			slog.Error("didn't expect a specific type (discord.RoleSelectMenuInteractionData)")
			return fmt.Errorf("cast error")
		}
		selectedRole = &casted.Values[0]
	case discord.ChannelSelectMenuInteractionData:
		casted, ok := data.(discord.ChannelSelectMenuInteractionData)
		if !ok {
			slog.Error("didn't expect a specific type (discord.ChannelSelectMenuInteractionData)")
			return fmt.Errorf("cast error")
		}
		selectedChannel = &casted.Values[0]
	}

	// handle diff acitons
	if selectedString != nil {
		switch *selectedString {
		case "LogChannel":
			menu := CreateChannelSelect(&selectRoute, 120, []string{event.User().String()}, "Choose a Channel...", handleGuildSettingsSelect)
			properties.NextAction = "LogChannel"
			replaceSelectInLookup(selectRoute, properties)
			sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User, menu)
		case "EnableVerifySystem", "DisableVerifySystem":
			values := make(map[string]bool)
			if *selectedString == "EnableVerifySystem" {
				values["verify_enabled"] = true
			} else {
				values["verify_enabled"] = false
			}
			db.Update(db.GetDB(), "guilds", values)
			slog.Info("guild's verify system was toggled", slog.String("guild", guild.ID.String()), slog.String("user", event.User().ID.String()), slog.Bool("value", values["verify_enabled"]))
			SendLog(Important, &guild.ID, "Verify System Togged", event.User().Mention()+" changed the guild's verify system to "+strconv.FormatBool(values["verify_enabled"]))
			sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User)
		case "VerifyRole":
			menu := CreateRoleSelect(&selectRoute, 120, []string{event.User().String()}, "Choose a Role...", handleGuildSettingsSelect)
			properties.NextAction = "VerifyRole"
			replaceSelectInLookup(selectRoute, properties)
			sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User, menu)
		case "VerifyChannel":
			menu := CreateChannelSelect(&selectRoute, 120, []string{event.User().String()}, "Choose a Channel...", handleGuildSettingsSelect)
			properties.NextAction = "VerifyChannel"
			replaceSelectInLookup(selectRoute, properties)
			sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User, menu)
		}
	} else if selectedRole != nil {
		switch properties.NextAction {
		case "VerifyRole":
			values := make(map[string]string)
			values["verify_role_id"] = selectedRole.String()
			db.Update(db.GetDB(), "guilds", values)
			slog.Info("guild's verify role was changed", slog.String("guild", guild.ID.String()), slog.String("user", event.User().ID.String()), slog.String("value", values["verify_role_id"]))
			SendLog(Important, &guild.ID, "Verify Role Changed", event.User().Mention()+" changed the guild's verify role to <@&"+selectedRole.String()+">")
		}
		sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User)
	} else if selectedChannel != nil {
		switch (*properties).NextAction {
		case "LogChannel":
			values := make(map[string]string)
			values["log_channel"] = selectedChannel.String()
			db.Update(db.GetDB(), "guilds", values)
			slog.Info("guild's log channel was changed", slog.String("guild", guild.ID.String()), slog.String("user", event.User().ID.String()), slog.String("value", values["log_channel"]))
			SendLog(Important, &guild.ID, "Log Channel Changed", event.User().Mention()+" changed the guild's log channel to <#"+selectedChannel.String()+">")
		case "VerifyChannel":
			values := make(map[string]string)
			values["channel_id"] = selectedChannel.String()
			db.Update(db.GetDB(), "guilds", values)
			slog.Info("guild's verify channel was changed", slog.String("guild", guild.ID.String()), slog.String("user", event.User().ID.String()), slog.String("value", values["channel_id"]))
			SendLog(Important, &guild.ID, "Verify Channel Changed", event.User().Mention()+" changed the guild's verify channel to <#"+selectedChannel.String()+">")
		}
		sendGuildSettingsMessage(event.Client().Rest, event.ApplicationID(), event.Token(), &guild.Guild, &event.Member().User)
	}

	return nil
}

type allowedSettingsType interface {
	string | int | bool | discord.Role | discord.GuildTextChannel
}

func getSettingFromDB[T allowedSettingsType](guild *discord.Guild, setting string) (*T, error) {
	var zero T

	// before doing anything, ensure the guild actually exists in the db
	db.EnsureGuildExists(db.GetDB(), guild.ID.String())

	// constriants
	constraints := make(map[string]string)
	constraints["id"] = guild.ID.String()

	// TODO: replace this giant switch statement with one check for an int to return and then have just one db query thing.
	switch any(zero).(type) {
	case string: // with strings and ints we can query directly into our helper function
		val, err := db.GetOne[string](db.GetDB(), "guilds", setting, constraints)
		if err != nil {
			slog.Error("couldn't fetch string from db in guild settings", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		return any(val).(*T), nil
	case int:
		val, err := db.GetOne[int](db.GetDB(), "guilds", setting, constraints)
		if err != nil {
			slog.Error("couldn't fetch int from db in guild settings", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, nil
		}
		if val == nil {
			return nil, nil
		}
		return any(val).(*T), nil
	case bool:
		val, err := db.GetOne[bool](db.GetDB(), "guilds", setting, constraints)
		if err != nil {
			slog.Error("couldn't fetch bool from db in guild settings", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, nil
		}
		if val == nil {
			return nil, nil
		}
		return any(val).(*T), nil
	case discord.Role:
		val, err := db.GetOne[string](db.GetDB(), "guilds", setting, constraints)
		if err != nil {
			slog.Error("couldn't fetch string from db in guild settings", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		// val would be a snowflake string id
		sf, err := snowflake.Parse(*val)
		if err != nil {
			slog.Error("couldn't convert string into a snowflake", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, err
		}

		role, err := client.Rest.GetRole(guild.ID, sf)
		if err != nil {
			slog.Error("couldn't get the member from snowflake id", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting), slog.String("snowflake", sf.String()))
			return nil, err
		}
		return any(role).(*T), nil
	case discord.GuildTextChannel:
		val, err := db.GetOne[string](db.GetDB(), "guilds", setting, constraints)
		if err != nil {
			slog.Error("couldn't fetch string from db in guild settings", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, err
		}
		if val == nil {
			return nil, nil
		}
		// val would be a snowflake string id
		sf, err := snowflake.Parse(*val)
		if err != nil {
			slog.Error("couldn't convert string into a snowflake", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting))
			return nil, err
		}

		channel, err := client.Rest.GetChannel(sf)
		if err != nil {
			slog.Error("couldn't get the guild text channel from snowflake id", slog.String("err", err.Error()), slog.String("guild", guild.ID.String()), slog.String("setting", setting), slog.String("snowflake", sf.String()))
			return nil, err
		}
		textChannel, ok := channel.(discord.GuildTextChannel)
		if !ok {
			slog.Error("couldn't cast discord channel into a guild text channel", slog.String("guild", guild.ID.String()), slog.String("setting", setting), slog.String("snowflake", sf.String()))
			return nil, fmt.Errorf("couldn't cast discord.Channel to a discord.GuildTextChannel")
		}
		return any(&textChannel).(*T), nil
	}
	return nil, nil
}

func getCurrentGuildSettings(guild *discord.Guild, author *discord.User) (*guildSettings, error) {
	result := &guildSettings{}
	selectMenuOptions := []SelectOptions{}
	var err error // dont let an error stop the rest of the errors

	selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Logging Channel", Value: "LogChannel", Description: "Logging channel for Isotope", Emoji: &discord.ComponentEmoji{Name: "✍️"}})
	if result.LogChannel, err = getSettingFromDB[discord.GuildTextChannel](guild, "log_channel"); err != nil {
		slog.Error("couldn't get log channel from db", slog.String("err", err.Error()))
		err = nil
	}

	if result.VerifySystem, err = getSettingFromDB[bool](guild, "verify_enabled"); err != nil {
		slog.Error("couldn't get verify enabled from the db", slog.String("err", err.Error()))
		err = nil
	}

	if *result.VerifySystem {
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Disable Verify System", Value: "DisableVerifySystem", Emoji: &discord.ComponentEmoji{Name: "🔴"}})

		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Verify Role", Value: "VerifyRole", Emoji: &discord.ComponentEmoji{Name: "✍️"}})
		if result.VerifyRole, err = getSettingFromDB[discord.Role](guild, "verify_role_id"); err != nil {
			slog.Error("couldn't get verify role from the db", slog.String("err", err.Error()))
			err = nil
		}

		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Change Verify Channel", Value: "VerifyChannel", Emoji: &discord.ComponentEmoji{Name: "✍️"}})
		if result.VerifyChannel, err = getSettingFromDB[discord.GuildTextChannel](guild, "channel_id"); err != nil {
			slog.Error("couldn't get verify channel from the db", slog.String("err", err.Error()))
			err = nil
		}
	} else {
		selectMenuOptions = append(selectMenuOptions, SelectOptions{Label: "Enable Verify System", Value: "EnableVerifySystem", Description: "Allows you to have Ion verification", Emoji: &discord.ComponentEmoji{Name: "🟢"}})
	}

	result.Select = CreateRestrictedSelect(nil, 120, []string{author.ID.String()}, "Change a Setting...", handleGuildSettingsSelect, selectMenuOptions...)

	return result, nil
}

func getCurrentGuildSettingsMessage(settings *guildSettings, guildName string) string {
	builder := GetNewMessageBuilder()
	builder.AddLargeHeader(guildName + " Settings")
	builder.AddSeperators()
	// log channel
	if settings.LogChannel != nil {
		builder.AddMessage("**Log Channel**: " + settings.LogChannel.Mention())
	} else {
		builder.AddMessage("**Log Channel**: Unset")
	}
	// verify system
	if settings.VerifySystem != nil {
		if *settings.VerifySystem {
			builder.AddMessage("**Verify System**: Enabled :white_check_mark:")
			// verify role
			if settings.VerifyRole != nil {
				builder.AddIndent("**Verify Role**: " + settings.VerifyRole.Mention())
			} else {
				builder.AddIndent("**Verify Role**: Unset")
			}
			// verify channel
			if settings.VerifyChannel != nil {
				builder.AddIndent("**Verify Channel**: " + settings.VerifyChannel.Mention())
			} else {
				builder.AddIndent("**Verify Channel**: Unset")
			}
		} else {
			builder.AddMessage("**Verify System**: Disabled :x:")
		}
	} else {
		builder.AddMessage("**Verify System**: Disabled :x: (data corrupted in database)")
	}

	return builder.BuildMessage()
}

func sendGuildSettingsMessage(rest rest.Rest, appID snowflake.ID, token string, guild *discord.Guild, author *discord.User, components ...discord.InteractiveComponent) error {
	var err error
	// get guild settings then the message
	settings, err := getCurrentGuildSettings(guild, author)
	if err != nil {
		slog.Error("couldn't get guild settings in sendInitalGuildSettingsMessage", slog.String("err", err.Error()))
		_, err = rest.UpdateInteractionResponse(appID, token, discord.NewMessageUpdate().WithContent("Internal Error."))
		return err
	}
	message := getCurrentGuildSettingsMessage(settings, guild.Name)
	messageUpdate := discord.NewMessageUpdate().WithContent(message).WithAllowedMentions(&discord.AllowedMentions{Parse: []discord.AllowedMentionType{}})

	// parse action rows
	const maxUnitsPerRow = 5
	unitsUsed := 0
	var currRow []discord.InteractiveComponent
	for _, component := range components {
		if component == nil {
			continue
		}

		// determine how much space it uses
		var itemUnit int
		switch component.Type() {
		case discord.ComponentTypeButton:
			itemUnit = 1
		case discord.ComponentTypeStringSelectMenu, discord.ComponentTypeUserSelectMenu, discord.ComponentTypeRoleSelectMenu, discord.ComponentTypeChannelSelectMenu:
			itemUnit = 5
		default:
			itemUnit = 5
		}

		// add action row depending on whether we consumed the entire row
		if unitsUsed+itemUnit > maxUnitsPerRow && len(currRow) > 0 {
			messageUpdate = messageUpdate.AddActionRow(currRow...)
			currRow = nil
			unitsUsed = 0
		}

		currRow = append(currRow, component)
		unitsUsed += itemUnit
	}

	// finish up the rest of the row
	if len(currRow) > 0 {
		messageUpdate = messageUpdate.AddActionRow(currRow...)
	}

	// if the action row is empty, add the default settings select
	if len(components) == 0 {
		messageUpdate = messageUpdate.AddActionRow(settings.Select)
	}

	_, err = rest.UpdateInteractionResponse(
		appID,
		token,
		messageUpdate,
	)
	if err != nil {
		slog.Error("couldn't update interaction message in sendGuildSettingsMessage", slog.String("err", err.Error()))
	}
	return err
}
