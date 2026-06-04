package commands

import (
	"log/slog"
	"maps"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
)

var r handler.Router
var cmds []discord.ApplicationCommandCreate
var tempCmdsStorage = make(map[string](func(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error))

func InitRouter() {
	r = handler.New()
	
	for _, route := range slices.Collect(maps.Keys(tempCmdsStorage)) {
		slog.Info("registering route " + route)
		r.SlashCommand(route, tempCmdsStorage[route])
	}

	client.EventManager.AddEventListeners(r)
	tempCmdsStorage = nil // force garbage collect
}

func RegisterCommand(cmdDetails discord.ApplicationCommandCreate, route string, cmdFunction func(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error) (error) {
	cmds = append(cmds, cmdDetails)
	tempCmdsStorage[route] = cmdFunction
	return nil
}

func SyncDev(guild *snowflake.ID) {
	handler.SyncCommands(GetClient(), cmds, []snowflake.ID{*guild})
}

func Sync() {
	handler.SyncCommands(GetClient(), cmds, []snowflake.ID{})
}