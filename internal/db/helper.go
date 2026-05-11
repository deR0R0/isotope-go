package db

import "database/sql"

func GetTokenData(db *sql.DB, userid string) (access_token string, token_type string, refresh_token string, expiry string, err error) {
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
