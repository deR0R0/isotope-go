package listeners

import (
	"log/slog"

	"github.com/deR0R0/isotope-go/internal/commands"
	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/disgoorg/disgo/events"
)

func OnMemberJoin(event *events.GuildMemberJoin) {
	userID := event.Member.User

	exists, err := db.UserExists(db.GetDB(), userID.ID.String())

	if err != nil {
		slog.Error("issue while getting person exists in db on member join", slog.String("err", err.Error()))
		return
	}

	if !exists {
		slog.Info("tried doing on memeber join event but they dont exist", slog.String("user", userID.Username))
		return
	}

	at, _, _, _, err := db.GetTokenData(db.GetDB(), userID.ID.String())

	if err != nil {
		slog.Error("error while getting user oauth data from db on member join", slog.String("err", err.Error()))
		return
	}

	if at == "" {
		// TODO: send logging!
		slog.Info("skipped automatic verification for user because they're not verified", slog.String("user", userID.Username))
		return 
	}

	// add role to the user
	verifyRole, err := commands.GetVerifyRoleFromGuild(&event.GuildID)
	if err != nil {
		slog.Error("error while getting verify role for guild", slog.String("guild", event.GuildID.String()))
		return
	}

	if verifyRole == nil {
		slog.Info("onmemberjoin guild doesn't have verify role set!", slog.String("guild", event.GuildID.String()))
		return
	}

	err = event.Client().Rest.AddMemberRole(event.GuildID, event.Member.User.ID, verifyRole.ID)

	if err != nil {
		slog.Error("there was an issue while trying to add the verified role for a guild", slog.String("guild", event.GuildID.String()), slog.String("user", event.Member.User.ID.String()))
		return
	}

	slog.Info("successfully automatically verified a user", slog.String("user", event.Member.User.ID.String()), slog.String("guild", event.GuildID.String()))
}