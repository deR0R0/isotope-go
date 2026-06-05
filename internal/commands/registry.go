package commands

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
)

var r handler.Router
var cmds []discord.ApplicationCommandCreate
var routes = []string{}
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

func RegisterCommand(cmdDetails discord.ApplicationCommandCreate, opts ...any) error {
	// route string, cmdFunction func(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error
	// ^ original parameters

	if len(opts) == 0 || len(opts)%2 != 0 {
		slog.Error("couldn't register command because it's in an improper format!", slog.String("name", cmdDetails.CommandName()))
		return fmt.Errorf("couldn't register command because it's in an improper format")
	}

	// each "optional parameter": route (string) -> cmdFunction (func()) -> repeat
	var route string
	for _, opt := range opts {
		switch v := opt.(type) {
		case string:
			route = v
		case (func(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error):
			tempCmdsStorage[route] = v
		default:
			panic("unknown data type provided in command register")
		}
	}

	cmds = append(cmds, cmdDetails)

	return nil
}

func ensureNoDuplicateRoutes(route string) {
	// don't register more than 1 route, otherwise handler will break
	idx := slices.Index(routes, route)
	if idx != -1 {
		// assume the current register is the superior one
		if route != "/button/isotope_authorize" { // hard code this permanent button for authorization/verification purposes
			slog.Info("route \"" + route + "\" already exists, removing previous...")
		}
		routes = slices.Delete(routes, idx, idx+1)
	}

	// register
	routes = append(routes, route)
}

func RegisterButton(route string, handlerFunc func(data discord.ButtonInteractionData, event *handler.ComponentEvent) error) {
	ensureNoDuplicateRoutes(route)

	// register the button
	r.ButtonComponent(route, handlerFunc)
}

func RegisterSelect(route string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error) {
	ensureNoDuplicateRoutes(route)

	// register the button
	r.SelectMenuComponent(route, handlerFunc)
}

func SyncDev(guild *snowflake.ID) {
	handler.SyncCommands(GetClient(), cmds, []snowflake.ID{*guild})
}

func Sync() {
	handler.SyncCommands(GetClient(), cmds, []snowflake.ID{})
}
