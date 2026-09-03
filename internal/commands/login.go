package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/deR0R0/isotope-go/internal/oauth"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

func init() {
	RegisterCommand(
		discord.SlashCommandCreate{
			Name:        "login",
			Description: "Log into your Ion account from this bot.",
		},
		"/login",
		loginCommandHandler,
	)
}

var LOGIN_LINK_TIMEOUT int = 10

type LoginResult struct {
	LoggedIn   bool
	Session    *oauth.Session
	ExpireTime int64
}

type LoginParams struct {
	UserID        string
	Rest          rest.Rest
	ApplicationID snowflake.ID
	Token         string
}

func login(userID string) (*LoginResult, error) {
	// check if user exists in the db and verify that they actually have data in access token
	exists, err := db.UserExists(db.GetDB(), userID)

	if err != nil {
		return nil, err
	}

	if exists {
		at, _, _, _, err := db.GetTokenData(db.GetDB(), userID)

		if err != nil {
			return nil, err
		}

		if at != "" {
			return &LoginResult{LoggedIn: true}, nil
		}
	}

	// grab oauth session
	session, err := oauth.Manager().CreateNewSession(userID)

	if err != nil {
		return nil, err
	}

	timeToExpire := time.Now().Unix() + int64(LOGIN_LINK_TIMEOUT)

	if !exists {
		db.CreateUser(db.GetDB(), userID)
	}

	db.SetState(db.GetDB(), userID, session.State)

	return &LoginResult{LoggedIn: false, Session: session, ExpireTime: timeToExpire}, nil
}

func HandleLogin(param *LoginParams) error {
	result, err := login(param.UserID)

	if err != nil {
		ShowErrorMessage("login", func() error {
			_, err := param.Rest.CreateFollowupMessage(
				param.ApplicationID,
				param.Token,
				discord.NewMessageCreate().WithContent("Internal Server Error").WithEphemeral(true),
			)
			return err
		})
	}

	if result.LoggedIn {
		param.Rest.CreateFollowupMessage(
			param.ApplicationID,
			param.Token,
			discord.NewMessageCreate().WithContent("You're already logged in!").WithEphemeral(true),
		)
		return err
	}

	msg, err := param.Rest.CreateFollowupMessage(
		param.ApplicationID,
		param.Token,
		discord.NewMessageCreate().WithContent("Hello! Here's your login link! Expires in <t:"+strconv.Itoa(int(result.ExpireTime))+":R>").WithComponents(discord.LayoutComponent(
			discord.NewActionRow(
				discord.NewLinkButton("Login", result.Session.RedirectURI),
				CreateNewButton("isotope_authorize", "Login", discord.ButtonStylePrimary, LoginButtonHandler),
			),
		)).WithEphemeral(true),
	)

	if err != nil {
		slog.Info("error while sending response: ", slog.Any("err", err))
		return err
	}

	DeleteAfter(time.Duration(LOGIN_LINK_TIMEOUT)*time.Second, func() error {
		return param.Rest.DeleteFollowupMessage(
			param.ApplicationID,
			param.Token,
			msg.ID,
		)
	})
	return nil
}

func loginCommandHandler(data discord.SlashCommandInteractionData, event *handler.CommandEvent) error {
	event.DeferCreateMessage(true)
	return HandleLogin(&LoginParams{UserID: event.User().ID.String(), Rest: event.Client().Rest, ApplicationID: event.ApplicationID(), Token: event.Token()})
}

func LoginButtonHandler(data discord.ButtonInteractionData, event *handler.ComponentEvent) error {
	event.DeferCreateMessage(true)
	return HandleLogin(&LoginParams{UserID: event.User().ID.String(), Rest: event.Client().Rest, ApplicationID: event.ApplicationID(), Token: event.Token()})
}

func HandleNewLogin(state string) error {
	// ensure this is actually their first time verifying
	userid, err := db.GetUserFromState(db.GetDB(), state)

	if err != nil {
		slog.Error("couldn't handle new login", slog.String("err", err.Error()))
		return err
	}

	slog.Info("handling a new login from user", slog.String("user", userid))

	first_time, err := db.GetFromUsers(db.GetDB(), userid, "firstTimeVerifying")

	ft, ok := first_time.(bool)
	if !ok {
		slog.Error("first_time return from the db is not bool type")
		return fmt.Errorf("Unexpected type returned from the DB.")
	}

	if !ft {
		slog.Info("not the user's first time verifying, skipping...")
		return nil
	}

	serversVerified := 0

	// first, process through the bot's cache for each guild
	for guild := range client.Caches.Guilds() {
		if err != nil {
			slog.Warn("couldn't get member from guild. most likely not in the guild ", slog.String("err", err.Error()))
			continue
		}

		role_id, _, err := db.GetGuildAuthData(db.GetDB(), guild.ID.String())

		if err != nil {
			slog.Error("err while getting guild auth data", slog.String("err", err.Error()))
			continue
		}

		if role_id == "" {
			slog.Warn("guild doesn't have a role set!", slog.String("guild", guild.ID.String()), slog.String("role", role_id))
			continue
		}

		err = AddRole(userid, role_id, guild.ID.String())
		if err != nil {
			slog.Error("err while adding role", slog.String("err", err.Error()))
			continue
		}

		serversVerified++
	}

	slog.Info("successfully verified user amount of guilds verified: ", slog.String("user", userid), slog.Int("guilds", serversVerified))

	err = db.SetToUsers(db.GetDB(), userid, "firstTimeVerifying", false)

	if err != nil {
		slog.Error("error setting first time verifying", slog.String("err", err.Error()))
	}

	return err
}
