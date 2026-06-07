package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"golang.org/x/oauth2"
)

/*
USER STUFF
*/

func UserExists(db *sql.DB, userid string) (bool, error) {
	var e bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userid).Scan(&e)

	if err != nil {
		return false, err
	}

	return e, nil
}

func CreateUser(db *sql.DB, userid string) error {
	_, err := db.Exec("INSERT INTO users (id, firstTimeVerifying) VALUES (?, ?)", userid, true)

	if err != nil {
		return err
	}

	return nil
}

func DeleteUser(db *sql.DB, userid string) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", userid)

	if err != nil {
		return err
	}

	return nil
}

func GetUserFromState(db *sql.DB, state string) (string, error) {
	var userid string
	err := db.QueryRow("SELECT id FROM users WHERE state = ?", state).Scan(&userid)

	if err != nil {
		return "", err
	}

	return userid, nil
}

// Users are returned by IDs
func GetAllUsers(db *sql.DB) (*[]string, error) {
	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var allIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		allIDs = append(allIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &allIDs, nil
}

/*
OAUTH HELPERS
*/

func GetTokenData(db *sql.DB, userid string) (string, string, string, string, error) {
	rows, err := db.Query(`SELECT access_token, token_type, refresh_token, expiry FROM users WHERE id = ?`, userid)

	if err != nil {
		return "", "", "", "", err
	}

	defer rows.Close()

	var at, tt, rt, e string

	for rows.Next() {
		rows.Scan(&at, &tt, &rt, &e)
	}

	return at, tt, rt, e, nil
}

func SetTokenData(db *sql.DB, userid string, at string, tt string, rt string, e string) error {
	_, err := db.Exec("UPDATE users SET access_token = ?, token_type = ?, refresh_token = ?, expiry = ? WHERE id = ?", at, tt, rt, e, userid)

	return err
}

func SaveTokenToDB(db *sql.DB, userid string, token *oauth2.Token) {
	SetTokenData(db, userid, token.AccessToken, token.TokenType, token.RefreshToken, token.Expiry.Format(time.RFC3339))
}

func GetState(db *sql.DB, userid string) (string, error) {
	var state string
	err := db.QueryRow("SELECT state FROM users WHERE id = ?", userid).Scan(&state)

	if err != nil {
		return "", err
	}

	return state, nil
}

func SetState(db *sql.DB, userid string, state string) error {
	_, err := db.Exec("UPDATE users SET state = ? WHERE id = ?", state, userid)

	if err != nil {
		return err
	}

	return nil
}

/*
GUILD STUFF
*/

func NewGuild(db *sql.DB, guildid string) error {
	_, err := db.Exec("INSERT INTO guilds (id, verify_enabled) VALUES (?, ?)", guildid, false)

	if err != nil {
		slog.Error("couldn't create a new guild", slog.String("err", err.Error()))
		return err
	}

	return nil
}

func GuildExists(db *sql.DB, guildid string) (bool, error) {
	var e bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM guilds WHERE id = ?)", guildid).Scan(&e)

	if err != nil {
		slog.Error("couldn't see if a guild exists", slog.String("err", err.Error()))
		return false, err
	}

	return e, nil
}

func DeleteGuild(db *sql.DB, guildid string) error {
	_, err := db.Exec("DELETE FROM guilds where id = ?", guildid)

	if err != nil {
		slog.Error("couldn't delete a guild", slog.String("err", err.Error()))
		return err
	}

	return nil
}

func EnsureGuildExists(db *sql.DB, guildid string) (error) {
	exists, err := GuildExists(db, guildid)

	if err != nil {
		return err
	}

	if !exists {
		err = NewGuild(db, guildid)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetAllGuilds(db *sql.DB) (*[]string, error) {
	rows, err := db.Query("SELECT id FROM guild")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var allIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		allIDs = append(allIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &allIDs, nil
}

/*
LOGIN/AUTHORZATION
*/

func GetGuildAuthData(db *sql.DB, guildid string) (string, string, error) {
	rows, err := db.Query("SELECT verify_role_id, channel_id FROM guilds WHERE id = ?", guildid)

	if err != nil {
		return "", "", err
	}

	defer rows.Close()

	var verify_role_id, auth_channel_id string

	for rows.Next() {
		rows.Scan(&verify_role_id, &auth_channel_id)
	}

	return verify_role_id, auth_channel_id, nil
}

/*
OTHER
*/

func GetFromUsers(db *sql.DB, userid string, key string) (any, error) {
	// sanitize input because we can't use ? placeholders in the sqlite3 thing
	allowedKeys := []string{"firstTimeVerifying"}

	if !slices.Contains(allowedKeys, key) {
		return nil, fmt.Errorf("unallowed key")
	}

	var value any
	err := db.QueryRow("SELECT "+key+" FROM users WHERE id = ?", userid).Scan(&value)

	if err != nil {
		return nil, err
	}

	return value, nil
}

func SetToUsers(db *sql.DB, userid string, key string, value any) error {
	allowedKey := []string{"firstTimeVerifying"}

	if !slices.Contains(allowedKey, key) {
		return fmt.Errorf("unallowed key")
	}

	_, err := db.Exec("UPDATE users SET "+key+" = ? WHERE id = ?", value, userid)

	return err
}

func GetFromGuilds(db *sql.DB, guildid string, key string) (any, error) {
	// sanitize input because we can't use ? placeholders in the sqlite3 thing
	allowedKeys := []string{"verify_role_id", "channel_id", "verify_enabled"}

	if !slices.Contains(allowedKeys, key) {
		return nil, fmt.Errorf("unallowed key")
	}

	err := EnsureGuildExists(db, guildid)

	if err != nil {
		slog.Error("couldn't ensure guild exists while getting from guild", slog.String("err", err.Error()))
	}

	var value any
	err = db.QueryRow("SELECT "+key+" FROM guilds WHERE id = ?", guildid).Scan(&value)

	if err != nil {
		return nil, err
	}

	return value, nil
}

func GetStringFromGuilds(db *sql.DB, guildid string, key string) (string, error) {
	value, err := GetFromGuilds(db, guildid, key)

	if err != nil {
		return "", err
	}

	if value == nil {
		return "", nil
	}

	v, ok := value.(string)

	if !ok {
		return "", fmt.Errorf("db returned type not string")
	}

	return v, nil
}

func GetBoolFromGuilds(db *sql.DB, guildid string, key string) (bool, error) {
	value, err := GetFromGuilds(db, guildid, key)

	if err != nil {
		return false, err
	}

	if value == nil {
		return false, fmt.Errorf("nil response")
	}

	v, ok := value.(bool)

	if !ok {
		return false, fmt.Errorf("db returned type not bool")
	}

	return v, nil
}

func SetToGuild(db *sql.DB, guildid string, key string, value any) error {
	allowedKeys := []string{"verify_role_id", "channel_id", "verify_enabled"}

	if !slices.Contains(allowedKeys, key) {
		return fmt.Errorf("unallowed key")
	}
	
	_, err := db.Exec("UPDATE guilds SET " + key + " = ? WHERE id = ?", value, guildid)

	return err
}