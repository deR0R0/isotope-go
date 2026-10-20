// this feature isn't necessarily a command,
// but i think putting logging here would
// be convienent

package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
)

type ImportanceLevel int

const (
	Normal ImportanceLevel = iota
	Important
	Warning
	Danger
)

func SendLog(importance ImportanceLevel, guild discord.Guild, header string, message string) {
	// send the log message
	channel, err := GetLogChannelFromGuild(&guild.ID)

	if err != nil {
		slog.Error("err while trying to send a log", slog.String("err", err.Error()))
		return
	}

	if channel == nil {
		slog.Info("channel nil")
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
	}
}
