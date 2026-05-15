package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/deR0R0/isotope-go/internal/db"
	"golang.org/x/oauth2"
)

var config *oauth2.Config = nil

type Session struct {
	State       string
	RedirectURI string
	Token       *oauth2.Token
}

type ManagerStruct struct {
	config   *oauth2.Config
	sessions map[string]*Session // <state>: <session_struct>
}

var manager *ManagerStruct // store a file-wide manager

func Init() {
	var redirect_uri string

	if os.Getenv("APP_ENV") == "production" {
		redirect_uri = os.Getenv("OAUTH_REDIR_URI")
	} else {
		redirect_uri = "http://localhost:" + os.Getenv("WEB_SERVER_PORT") + "/login"
	}

	config = &oauth2.Config{
		ClientID:     os.Getenv("ION_CLIENT_ID"),
		ClientSecret: os.Getenv("ION_SECRET_ID"),
		Scopes:       []string{"read", "write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv("OAUTH_AUTH_URL"),
			TokenURL: os.Getenv("OAUTH_TOKEN_URL"),
		},
		RedirectURL: redirect_uri,
	}

	manager = &ManagerStruct{
		config:   config,
		sessions: make(map[string]*Session),
	}

	// from the database, get all the users and tokens n stuff and resume their session.
	users, err := db.GetAllUsers(db.GetDB())
	if err != nil {
		slog.Error("couldn't get all users", slog.String("err", err.Error()))
		panic("can't resume oauth sessions")
	}

	database := db.GetDB()

	for _, user := range *users {
		at, tt, rt, e, err := db.GetTokenData(database, user)

		if err != nil {
			slog.Warn("failed to get token data from user", slog.String("id", user), slog.String("err", err.Error()))
			continue
		}

		if at == "" {
			continue // skip, they have no access token set yet
		}

		// resume the user's session

		// state
		var state string
		if state, err = generateState(); err != nil {
			slog.Warn("couldn't generate a state", slog.String("err", err.Error()))
			continue
		}

		// auth uri
		var uri string
		if uri, err = manager.getAuthURI(&state); err != nil {
			slog.Warn("couldn't get auth uri", slog.String("err", err.Error()))
			continue
		}

		// parse expiry
		var expiry time.Time
		if expiry, err = time.Parse(time.RFC3339, e); err != nil {
			slog.Warn("couldn't get time", slog.String("expiry", e), slog.String("err", err.Error()))
			continue
		}

		// create our token
		token := &oauth2.Token{
			AccessToken:  at,
			TokenType:    tt,
			RefreshToken: rt,
			Expiry:       expiry,
		}

		s := &Session{
			state,
			uri,
			token,
		}

		slog.Info("successfully resumed user", slog.String("id", user))

		manager.sessions[user] = s
	}
}

func Manager() *ManagerStruct {
	return manager
}

func (m *ManagerStruct) getAuthURI(state *string) (string, error) {
	if state == nil {
		return "", fmt.Errorf("state is empty")
	}

	return m.config.AuthCodeURL(*state, oauth2.AccessTypeOffline), nil
}

func generateState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (m *ManagerStruct) CreateNewSession(userID string) (*Session, error) {
	state, err := generateState()

	if err != nil {
		slog.Error("issue while generating state")
		return nil, err
	}

	// generate the auth uri
	authURI, err := m.getAuthURI(&state)

	if err != nil {
		slog.Error("issue while generating uri")
		return nil, err
	}

	// actually create our session. token is nil until we get our token
	session := &Session{
		State:       state,
		RedirectURI: authURI,
		Token:       nil,
	}

	// add the session to our manager
	m.sessions[userID] = session

	return session, nil
}

func (m *ManagerStruct) GetSession(userID string) (*Session, error) {
	session, ok := m.sessions[userID]

	if !ok {
		return nil, fmt.Errorf("couldn't get session")
	}

	return session, nil
}

func (m *ManagerStruct) GetSessionFromState(state string) (*Session, error) {
	for _, session := range m.sessions {
		if session.State == state {
			return session, nil
		}
	}

	return nil, fmt.Errorf("couldn't find session")
}

func (m *ManagerStruct) DeleteSession(userID string) {
	delete(m.sessions, userID)
}

func (m *ManagerStruct) SetToken(state *string, token *oauth2.Token) error {
	// O(n) instead of O(1) consider optimizing
	for _, session := range m.sessions {
		if session.State == *state {
			session.Token = token

			// run a goroutine to save it to the db!
			go func() {
				userid, err := db.GetUserFromState(db.GetDB(), *state)

				if err != nil {
					slog.Error("couldn't save token to db!", slog.String("err", err.Error()))
					return
				}

				db.SaveTokenToDB(db.GetDB(), userid, token)
			}()

			return nil
		}
	}
	return fmt.Errorf("couldn't find state")
}

// functions that make it easier to work with oauth
func (m *ManagerStruct) Exchange(ctx context.Context, code string, state string) error {
	token, err := m.config.Exchange(ctx, code)

	if err != nil {
		return err
	}

	err = m.SetToken(&state, token)
	if err != nil {
		return err
	}

	return nil
}
