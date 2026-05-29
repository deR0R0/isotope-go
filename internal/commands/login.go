package commands

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/deR0R0/isotope-go/internal/oauth"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	Register(discord.SlashCommandCreate{
		Name:        "login",
		Description: "Log into your Ion account from this bot.",
	})
}

var LOGIN_LINK_TIMEOUT int = 10

// local error message to make it quick!
func loginShowErrorMessage(event *events.ApplicationCommandInteractionCreate) {
	ShowErrorMessage("login", func() error {
		_, err := event.Client().Rest.CreateFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			discord.NewMessageCreate().WithContent("Internal Server Error"),
		)
		return err
	})
}

func handleLogin(event *events.ApplicationCommandInteractionCreate) {
	// first, defer the interaction. show the "<bot> is thinking..." message (this may be a long-running task)
	event.DeferCreateMessage(true)

	userID := event.User().ID.String()

	// check if user exists in the db and verify that they actually have data in access token
	exists, err := db.UserExists(db.GetDB(), userID)
	if exists {
		at, _, _, _, err := db.GetTokenData(db.GetDB(), userID)

		if err != nil {
			loginShowErrorMessage(event)
			return
		}

		if at != "" {
			event.Client().Rest.CreateFollowupMessage(
				event.ApplicationID(),
				event.Token(),
				discord.NewMessageCreate().WithContent("You're already logged in!"),
			)
			return
		}
	}

	session, err := oauth.Manager().CreateNewSession(event.User().ID.String())

	if err != nil {
		loginShowErrorMessage(event)
		return
	}

	timeToExpire := time.Now().Unix() + int64(LOGIN_LINK_TIMEOUT)

	msg, err := event.Client().Rest.CreateFollowupMessage(
		event.ApplicationID(),
		event.Token(),
		discord.NewMessageCreate().WithContent("Hello! Here's your login link! Expires in <t:"+strconv.Itoa(int(timeToExpire))+":R>").WithComponents(discord.LayoutComponent(
			discord.NewActionRow(
				discord.NewLinkButton("Login", session.RedirectURI),
			),
		)),
	)

	if err != nil {
		slog.Info("error while sending response: ", slog.Any("err", err))
		return
	}

	DeleteAfter(time.Duration(LOGIN_LINK_TIMEOUT)*time.Second, func() error {
		return event.Client().Rest.DeleteFollowupMessage(
			event.ApplicationID(),
			event.Token(),
			msg.ID,
		)
	})

	if !exists {
		db.CreateUser(db.GetDB(), userID)
	}

	db.SetState(db.GetDB(), userID, session.State)
}

func HandleNewLogin(state string) error {
	// ensure this is actually their first time verifying
	userid, err := db.GetUserFromState(db.GetDB(), state)

	if err != nil {
		slog.Error("couldn't handle new login", slog.String("err", err.Error()))
		return err
	}

	slog.Info("handling a new login from user", slog.String("user", userid))

	first_time, err := db.Get(db.GetDB(), userid, "firstTimeVerifying")

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

	err = db.Set(db.GetDB(), userid, "firstTimeVerifying", false)

	if err != nil {
		slog.Error("error setting first time verifying", slog.String("err", err.Error()))
	}

	return err
}
