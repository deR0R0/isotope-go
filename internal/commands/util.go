package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

var client *bot.Client

func SetClient(c *bot.Client) {
	client = c
}

func GetClient() *bot.Client {
	return client
}

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
