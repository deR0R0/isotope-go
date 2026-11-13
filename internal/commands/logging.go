// this feature isn't necessarily a command,
// but i think putting logging here would
// be convienent

package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type ImportanceLevel int

const (
	Normal ImportanceLevel = iota
	Important
	Warning
	Danger
)

func SendLog(importance ImportanceLevel, guild *snowflake.ID, header string, message string) {
	// get guild
	g, err := client.Rest.GetGuild(*guild, false)
	// send the log message
	channel, err := getSettingFromDB[discord.GuildTextChannel](&g.Guild, "log_channel")

	if err != nil {
		slog.Error("err while trying to send a log", slog.String("err", err.Error()))
		return
	}

	if channel == nil {
		slog.Info("attempted to send a log to guild, but channel was nil. ignoring.", slog.String("guild", guild.String()))
		return
	}

	logMessage := GetNewMessageBuilder()

	importanceEmoji := "⬜️"

	switch importance {
	case Important:
		importanceEmoji = "🟨"
	case Warning:
		importanceEmoji = "⚠️"
	case Danger:
		importanceEmoji = "⚠️⚠️⚠️" // very creative i know
	}

	logMessage.AddMessage(importanceEmoji + " **" + header + "** " + importanceEmoji)
	logMessage.AddMessage(message)

	_, err = client.Rest.CreateMessage(
		(*channel).ID(),
		discord.NewMessageCreate().
			WithContent(logMessage.BuildMessage()).
			WithAllowedMentions(&discord.AllowedMentions{
				Parse: []discord.AllowedMentionType{},
			}),
	)

	if err != nil {
		slog.Error("err while sending a log message", slog.String("err", err.Error()))
	} else {
		slog.Info("sent a log message to a guild", slog.String("guild", guild.String()), slog.String("header", header), slog.String("message", message))
	}
}
