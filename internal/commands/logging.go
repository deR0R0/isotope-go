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
)

func SendLog(guild discord.Guild, importance ImportanceLevel, message string, author *discord.User, opts ...discord.EmbedField) {
	// build embed
	logEmbed := discord.Embed{}.WithTitle(message)
	
	switch importance {
	case Normal:
		logEmbed = logEmbed.WithColor(0x4287f5)
	case Important:
		logEmbed = logEmbed.WithColor(0xebdc10)
	}
	
	if author != nil {
		logEmbed = logEmbed.WithAuthor(author.Username, "", *(*author).AvatarURL())
	}

	for _, opt := range opts {
		logEmbed = logEmbed.AddField(opt.Name, opt.Value, true)
	}
	
	// send the embed
	channel, err := GetLogChannelFromGuild(&guild.ID)
	
	if err != nil {
		slog.Error("err while trying to send a log", slog.String("err", err.Error()))
		return
	}
	
	if channel == nil {
		slog.Info("channel nil")
		return
	}
	
	_, err = client.Rest.CreateMessage((*channel).ID(), discord.NewMessageCreate().WithEmbeds(logEmbed).WithAllowedMentions(&discord.AllowedMentions{Parse: []discord.AllowedMentionType{}}))

	if err != nil {
		slog.Error("err while sending a log message", slog.String("err", err.Error()))
	}
}