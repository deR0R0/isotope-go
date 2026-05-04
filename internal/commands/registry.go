package commands

import (
	"log/slog"

	"github.com/disgoorg/disgo/discord"
)

var Commands []discord.ApplicationCommandCreate

func Register(cmd discord.ApplicationCommandCreate) {
	slog.Info("registered command ", slog.String("name", cmd.CommandName()))
	Commands = append(Commands, cmd)
}