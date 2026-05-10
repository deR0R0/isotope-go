package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB_NAME = "store.db"
var db *sql.DB = nil

var migrations = []string {
	// v1, schema:
	/*
		users:
			userId - primary key, discord id
			state - store oauth state
			accesstoken - part of token
			tokentype - part of token
			refreshtoken - part of token
			expiry - text but a date string, part of token
		guilds:
			guildID - primary key, int
			verifyRoleID - role to give when authentication successful
			channelID - channel id where to put the button
			
	*/
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER NOT NULL PRIMARY KEY,
		state TEXT,
		access_token TEXT,
		token_type TEXT,
		refresh_token TEXT,
		expiry TEXT
	);
	CREATE TABLE IF NOT EXISTS guilds (
		id INTEGER NOT NULL PRIMARY KEY,
		verify_role_id INTEGER,
		channel_id INTEGER
	);
	`,
}

func migrateDB(database *sql.DB) (error) {
	// query for the db version and compare with migration length
	var db_version int
	err := database.QueryRow("PRAGMA user_version").Scan(&db_version)
	if err != nil {
		slog.Error("could not migrate db due to getting db version issue")
		return err
	}

	if db_version != len(migrations) {
		slog.Info("db version not same as migration version. migrating... ", slog.Int("db_version", db_version), slog.Int("migration_version", len(migrations)))
		// not equal to migrations, we shall migrate our db up to that point
		var err error;
		for i := db_version; i < len(migrations); i++ { // start i at database version
			_, err = database.Exec(migrations[i])
			if err != nil {
				slog.Error("oops, issue while migrating the db. exiting...", slog.String("err", err.Error()))
				panic("cannot migrate db")
			}
		}
	} else {
		slog.Info("database schema is up to date. skipping migration.")
		return nil
	}

	slog.Info("successfully migrated the db. bumping version up", slog.Int("version", len(migrations)))
	_, err = db.Exec(`PRAGMA user_version = ` + fmt.Sprintf("%d", len(migrations)))
	if err != nil {
		slog.Error("couldn't bump user version of the db")
		return err
	}

	return nil
}

func provisionDB(database *sql.DB, first_boot bool) {
	// if it's not our first boot, close connection and rename the previous db
	if !first_boot {
		database.Close()
		os.Rename("./" + DB_NAME, "./old_" + DB_NAME)
	}

	var err error
	database, err = sql.Open("sqlite3", "./" + DB_NAME)
	if err != nil {
		slog.Error("cannot open database file. fix permissions and rerun. exiting now.")
		panic("database open err")
	}

	slog.Info("provisined the db")
}

func Init() {
	var first_boot = false
	if _, err := os.Stat("./" + DB_NAME); errors.Is(err, os.ErrNotExist) {
		first_boot = true
	} else if err != nil {
		slog.Error("cannot access the database file (may be permission denied). fix permissions and rerun. exiting now.", slog.String("err", err.Error()))
		panic("database error")
	}

	var err error
	db, err = sql.Open("sqlite3", "./" + DB_NAME)
	if err != nil {
		slog.Error("couldn't open db file. oh no. fix permissions please", slog.String("err", err.Error()))
	}

	if first_boot {
		provisionDB(db, first_boot)
	}
	migrateDB(db)
}