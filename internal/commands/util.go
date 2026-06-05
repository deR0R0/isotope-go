package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
)

/* CLIENT STUFF */
var client *bot.Client

func SetClient(c *bot.Client) {
	client = c
}

func GetClient() *bot.Client {
	return client
}

/* Discord Message Stuff */

func DeleteAfter(delay time.Duration, deleteFunction func() error) {
	go func() {
		time.Sleep(delay)
		if err := deleteFunction(); err != nil {
			slog.Error("err while deleting message, possibly safe to ignore", slog.String("err", err.Error()))
		}
	}()
}

func ShowErrorMessage(source string, editFunction func() error) {
	slog.Error(source + " had an error. giving the user response message")
	if err := editFunction(); err != nil {
		slog.Error("wow, another err while editing the function.")
	}
}

/* Discord Role Helpers */

func AddRole(userid string, roleid string, guildid string) error {
	// get the snowflakes
	userSnowflake := snowflake.MustParse(userid)
	roleSnowflake := snowflake.MustParse(roleid)
	guildSnowflake := snowflake.MustParse(guildid)

	// use rest api to retreieve the actual objects
	var err error
	var guild *discord.RestGuild
	var role *discord.Role

	if guild, err = client.Rest.GetGuild(guildSnowflake, true); err != nil {
		return err
	}

	if role, err = client.Rest.GetRole(guildSnowflake, roleSnowflake); err != nil {
		return err
	}

	// ensure the role exists in the guild
	if !slices.Contains(guild.Roles, *role) {
		slog.Error("guild does't have the role anymore.")
		// TODO: clear the db of this role if the bot can't find it
		return fmt.Errorf("guild doesn't have role")
	}

	// finally add the role
	if err = client.Rest.AddMemberRole(guildSnowflake, userSnowflake, roleSnowflake); err != nil {
		return err
	}

	return nil
}

/* Router Component Helpers */

func CreateNewButton(id string, label string, style discord.ButtonStyle, handlerFunc (func(data discord.ButtonInteractionData, event *handler.ComponentEvent) error)) *discord.ButtonComponent {
	route := "/button/" + id

	// don't use the NewPrimaryButton bs, just directly create the struct
	button := discord.ButtonComponent{
		Style: style,
		Label: label,
		CustomID: route,
	}

	slog.Info("registering button under route " + route, slog.String("id", id), slog.String("label", label))
	RegisterButton(route, handlerFunc)
	return &button
}

func CreateNewSelect(id string, placeholder string, handlerFunc (func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error), opts ...string) *discord.StringSelectMenuComponent {
	route := "/select/" + id

	// parse ze options
	options := make([]discord.StringSelectMenuOption, len(opts))
	for i, opt := range opts {
		options[i] = discord.NewStringSelectMenuOption(opt, strings.ToLower(opt))
	}

	// build menu
	menu := discord.NewStringSelectMenu(
		route,
		placeholder,
		options...
	)
	
	slog.Info("registering string select menu under route " + route, slog.String("id", id))
	RegisterSelect(route, handlerFunc)
	
	return &menu
}