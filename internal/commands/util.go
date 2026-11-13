package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/lithammer/shortuuid/v5"
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
			slog.Warn("oops, you messed up formatting. here's the element type", slog.Int("type", int(element.sectionType)))
			output += "You aren't supposed to see this.\n"
		}
	}

	return output
}

/* Discord Role Helpers */

// compares two roles to see if they're the same role. (normal comparisons compare all fields, this compares only ID)
func IsSameRole(role1 discord.Role, role2 discord.Role) bool {
	return role1.ID == role2.ID
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
	foundRole := false
	for _, guildRole := range guild.Roles {
		if IsSameRole(guildRole, *role) {
			foundRole = true
			break
		}
	}

	if !foundRole {
		slog.Info("couldn't find the role you're trying to add in the guild's roles", slog.String("guild_id", guildid), slog.String("role_id", roleid))
		return fmt.Errorf("couldn't find requested role in guild")
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

func GetVerifyChannelFromGuild(guild *snowflake.ID) (*discord.Channel, error) {
	channelID, err := db.GetStringFromGuilds(db.GetDB(), guild.String(), "channel_id")
	if err != nil {
		return nil, err
	}

	if channelID == "" {
		return nil, nil
	}

	sf, err := snowflake.Parse(channelID)
	if err != nil {
		return nil, err
	}

	// grab channel from discord - why does it return the actual value rather than a pointer?
	channel, err := client.Rest.GetChannel(sf)
	if err != nil {
		return nil, err
	}

	return &channel, nil
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

/*
SELECT MENU STUFF
*/

type SelectOptions struct {
	Label       string
	Value       string
	Description string
	Emoji       *discord.ComponentEmoji
}

// the "settings" that the lookup to select menu would have
type SelectMenuProperties struct {
	//Select *discord.StringSelectMenuComponent <- i currently do not see a need for this field rn. possibly needed later and NOTE: use discord.SelectMenuComponent instead as the common "inheritance" of all types
	Expiry     time.Time
	Users      []string // strings of user ids because i literally can't do a slice contain with normal discord.Users
	NextAction string
}

type SelectMenuLookup struct {
	Keys map[string]SelectMenuProperties
	sync.RWMutex
}

var selectLookup SelectMenuLookup

func addSelectToLookup(id string, expiry time.Time, users []string) {
	// use rw mutex lock to prevent concurrent map write panic (whoops)
	selectLookup.RWMutex.Lock()
	defer selectLookup.Unlock()
	if selectLookup.Keys == nil {
		selectLookup.Keys = make(map[string]SelectMenuProperties)
	}
	selectLookup.Keys[id] = SelectMenuProperties{Expiry: expiry, Users: users}
}

func replaceSelectInLookup(id string, properties *SelectMenuProperties) {
	selectLookup.RWMutex.Lock()
	defer selectLookup.Unlock()
	selectLookup.Keys[id] = *properties
}

func getSelectFromLookup(id string) *SelectMenuProperties {
	selectLookup.RWMutex.RLock()
	defer selectLookup.RUnlock()
	if selectLookup.Keys == nil {
		selectLookup.Keys = make(map[string]SelectMenuProperties)
	}
	// yoink it
	menu, ok := selectLookup.Keys[id]
	if !ok {
		return nil
	}
	return &menu
}

// this is to create a permanent select
func CreateNewSelect(id string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error, opts ...SelectOptions) (string, *discord.StringSelectMenuComponent) {
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
		id,
		placeholder,
		options...,
	)

	slog.Info("registering string select menu under route "+id, slog.String("id", id))
	RegisterSelect(id, handlerFunc)

	return id, &menu
}

// create select menu but with other restrictions. also, BIG FUNCTION
func CreateRestrictedSelect(
	id *string,
	expireSeconds int64,
	users []string,
	placeholder string,
	handlerFunc func(data discord.SelectMenuInteractionData,
		event *handler.ComponentEvent) error,
	opts ...SelectOptions) *discord.StringSelectMenuComponent {
	if id == nil {
		u := shortuuid.New() // use this library instead of the default uuid, default too long and may overflow. this is 22 characters (?) so its good
		u = "/select/" + u
		id = &u
	}
	route, menu := CreateNewSelect(*id, placeholder, handlerFunc, opts...)
	addSelectToLookup(route, time.Now().Add(time.Duration(expireSeconds)), users)
	return menu
}

// TODO: split CreateRoleSelect (more options, universal) and CreateRestrictedRoleSelect
// creates a special discord role select
func CreateRoleSelect(id *string, expireSeconds int64, users []string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error) *discord.RoleSelectMenuComponent {
	if id == nil {
		u := shortuuid.New()
		u = "/select/" + u
		id = &u
	}

	// create a menu with only 1 option available. can later seperate this logic if required.
	menu := discord.NewRoleSelectMenu(
		*id,
		placeholder,
	).WithMinValues(1).WithMaxValues(1)

	slog.Info("registering new role select menu under route "+*id, slog.String("id", *id))
	RegisterSelect(*id, handlerFunc)
	addSelectToLookup(*id, time.Now().Add(time.Duration(expireSeconds)), users)

	return &menu
}

// creates a special discord channel select
func CreateChannelSelect(id *string, expireSeconds int64, users []string, placeholder string, handlerFunc func(data discord.SelectMenuInteractionData, event *handler.ComponentEvent) error) *discord.ChannelSelectMenuComponent {
	if id == nil {
		u := shortuuid.New()
		u = "/select/" + u
		id = &u
	}

	// create a menu with only 1 option available. can later seperate this logic if required.
	menu := discord.NewChannelSelectMenu(
		*id,
		placeholder,
	).WithMinValues(1).WithMaxValues(1).WithChannelTypes(discord.ChannelTypeGuildText)

	slog.Info("registering new channel select menu under route "+*id, slog.String("id", *id))
	RegisterSelect(*id, handlerFunc)
	addSelectToLookup(*id, time.Now().Add(time.Duration(expireSeconds)), users)

	return &menu
}
