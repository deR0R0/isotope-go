package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var API_ROUTE_BASE = "https://ion.tjhsst.edu/api"
var API_ROUTE_SCHEDULE string = "/schedule"
var API_ROUTE_EMERG string = "/emerg"
var API_ROUTE_PROFILE string = "/profile"
var API_ROUTE_BLOCKS string = "/blocks"
var API_ROUTE_ACTIVITIES string = "/activities"
var API_ROUTE_SIGNUPS string = "/signups"

// this function is NOT responsible for parsing json from the api.
func (m *ManagerStruct) get(userid string, url string) (*http.Response, error) {
	session, ok := m.sessions[userid]

	if !ok {
		return nil, fmt.Errorf("userid doesn't exist")
	}

	if !session.Token.Valid() {
		return nil, fmt.Errorf("user has invalid token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := m.config.Client(ctx, session.Token)

	return client.Get(url)
}

func (m *ManagerStruct) GetProfile(userid string) (Profile, error) {
	res, err := m.get(userid, API_ROUTE_BASE+API_ROUTE_PROFILE)
	var profileRes Profile

	if err != nil {
		slog.Error("error while getting profile", slog.String("user", userid), slog.String("err", err.Error()))
		return profileRes, err
	}

	// parse the body in json
	if err = json.NewDecoder(res.Body).Decode(&profileRes); err != nil {
		slog.Error("err while parsing json", slog.String("user", userid), slog.String("err", err.Error()))
		return profileRes, err
	}

	return profileRes, nil
}
