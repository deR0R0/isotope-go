package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
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

/* A thing that helps with message building */
type MessageBuilder struct {
	sections []MessageBuilderSection
}

type MessageBuilderSectionType int

const (
	SectionTypeMessage MessageBuilderSectionType = iota
	SectionTypeLargeHeader
	SectionTypeMediumHeader
	SectionTypeSmallHeader
	SectionTypeIndent
	SectionTypeCodeBlock
	SectionTypeSeperator
)

type MessageBuilderSection struct {
	message     string
	sectionType MessageBuilderSectionType
}

func GetNewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

func (mb *MessageBuilder) AddLargeHeader(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeLargeHeader})
}

func (mb *MessageBuilder) AddMediumHeader(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeMediumHeader})
}

func (mb *MessageBuilder) AddSmallHeader(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeSmallHeader})
}

func (mb *MessageBuilder) AddMessage(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeMessage})
}

func (mb *MessageBuilder) AddIndent(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeIndent})
}

func (mb *MessageBuilder) AddCodeBlock(msg string) {
	mb.sections = append(mb.sections, MessageBuilderSection{message: msg, sectionType: SectionTypeCodeBlock})
}

func (mb *MessageBuilder) AddSeperators() {
	mb.sections = append(mb.sections, MessageBuilderSection{sectionType: SectionTypeSeperator})
}

func (mb *MessageBuilder) BuildMessage() string {
	// this method is to build the message from the message builder (no shit sherlock)
	var output string = ""
	for _, element := range mb.sections {
		switch element.sectionType {
		case SectionTypeMessage:
			output += element.message + "\n"
		case SectionTypeLargeHeader:
			output += "# " + element.message + "\n"
		case SectionTypeMediumHeader:
			output += "## " + element.message + "\n"
		case SectionTypeSmallHeader:
			output += "### " + element.message + "\n"
		case SectionTypeIndent:
			output += "> " + element.message + "\n"
		case SectionTypeCodeBlock:
			output += "`" + element.message + "`\n"
		case SectionTypeSeperator:
			output += "\n\u200B"
		default:
			output += "You aren't supposed to see this.\n"
		}
	}

	return output
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

/* I dunno what this is */
func GetVerifyRoleFromGuild(guild *snowflake.ID) (*discord.Role, error) {
	roleid, err := db.GetStringFromGuilds(db.GetDB(), guild.String(), "verify_role_id")
	if err != nil {
		return nil, err
	}

	// empty role - not set
	if roleid == "" {
		return nil, nil
	}

	sf, err := snowflake.Parse(roleid)
	if err != nil {
		return nil, err
	}

	// grab role from discord
	role, err := client.Rest.GetRole(*guild, sf)
	if err != nil {
		return nil, err
	}

	return role, nil
}

/* Router Component Helpers */

func CreateNewButton(id string, label string, style discord.ButtonStyle, handlerFunc func(data discord.ButtonInteractionData, event *handler.ComponentEvent) error) *discord.ButtonComponent {
	route := "/button/" + id

	// don't use the NewPrimaryButton bs, just directly create the struct
	button := discord.ButtonComponent{
		Style:    style,
		Label:    label,
		CustomID: route,
	}

	slog.Info("registering button under route "+route, slog.String("id", id), slog.String("label", label))
	RegisterButton(route, handlerFunc)
	return &button
}

func CreateNewRestrictedButton(expireSeconds int64, userID snowflake.ID, id string, label string, style discord.ButtonStyle, handlerFunc func(data discord.ButtonInteractionData, event *handler.ComponentEvent) error) *discord.ButtonComponent {
	route := HandlerRoute{}
	route.SetBase(id)
	route.AddRestrictor("expire", strconv.FormatInt(time.Now().Unix(), 10))
	route.AddRestrictor("author", strconv.FormatUint(uint64(userID), 10))
	return CreateNewButton(route.GetRoute(), label, style, handlerFunc)
}

/* select */

type SelectOptions struct {
	Label       string
	Value       string
	Description string
	Emoji       *discord.ComponentEmoji
}

// this is to create a permanent select
func CreateNewSelect(id string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error, opts ...SelectOptions) *discord.StringSelectMenuComponent {
	route := "/select/" + id

	// parse ze options
	// opts = param
	// opt = singular option
	// option = select menu option
	options := make([]discord.StringSelectMenuOption, len(opts))
	for i, opt := range opts {
		if opt.Label == "" || opt.Value == "" {
			slog.Warn("skipping a select menu option because it's missing either a label or a value.", slog.String("id", id), slog.String("label", opt.Label), slog.String("value", opt.Value))
			continue
		}
		option := discord.NewStringSelectMenuOption(opt.Label, opt.Value).WithDescription(opt.Description)
		option.Emoji = opt.Emoji
		options[i] = option
	}

	// build menu
	menu := discord.NewStringSelectMenu(
		route,
		placeholder,
		options...,
	)

	slog.Info("registering string select menu under route "+route, slog.String("id", id))
	RegisterSelect(route, handlerFunc)

	return &menu
}

// creates a select but with the proper expiry and id.
func CreateRestrictedSelect(expireSeconds int64, userID snowflake.ID, id string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error, opts ...SelectOptions) *discord.StringSelectMenuComponent {
	route := HandlerRoute{}
	route.SetBase(id)
	route.AddRestrictor("expire", strconv.FormatInt(time.Now().Unix(), 10))
	route.AddRestrictor("author", strconv.FormatUint(uint64(userID), 10))
	return CreateNewSelect(route.GetRoute(), placeholder, handlerFunc, opts...)
}

// TODO: split CreateRoleSelect (more options, universal) and CreateRestrictedRoleSelect
// creates a special discord role select
func CreateRoleSelect(expireSeconds int64, userID snowflake.ID, id string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error) *discord.RoleSelectMenuComponent {
	route := HandlerRoute{}
	route.SetBase(id)
	route.AddRestrictor("expire", strconv.FormatInt(time.Now().Unix(), 10))
	route.AddRestrictor("author", strconv.FormatUint(uint64(userID), 10))

	finalRoute := "/select/" + route.GetRoute()

	// create a menu with only 1 option available. can later seperate this logic if required.
	menu := discord.NewRoleSelectMenu(
		finalRoute,
		placeholder,
	).WithMinValues(1).WithMaxValues(1)

	slog.Info("registering new role select menu under route "+finalRoute, slog.String("id", id))
	RegisterRoleSelect(finalRoute, handlerFunc)

	return &menu
}
