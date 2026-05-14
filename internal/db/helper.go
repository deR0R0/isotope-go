package db

import (
	"database/sql"
)

func UserExists(db *sql.DB, userid string) (bool, error) {
	var e bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userid).Scan(&e)

	if err != nil {
		return false, err
	}

	return e, nil
}

func CreateUser(db *sql.DB, userid string) error {
	_, err := db.Exec("INSERT INTO users (id) VALUES (?)", userid)

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
