package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/oauth2"
)

var config *oauth2.Config = nil

type Session struct {
	State string
	RedirectURI string
	Token *oauth2.Token
}

type ManagerStruct struct {
	config *oauth2.Config
	sessions map[string]*Session // <state>: <session_struct>
}

var manager *ManagerStruct // store a file-wide manager

func init() {
	config = &oauth2.Config{
		ClientID: os.Getenv("ION_CLIENT_ID"),
		ClientSecret: os.Getenv("ION_SECRET_ID"),
		Scopes: []string{"read", "write"},
		Endpoint: oauth2.Endpoint{
			AuthURL: os.Getenv("OAUTH_AUTH_URL"),
			TokenURL: os.Getenv("OAUTH_TOKEN_URL"),
		},
	}
	
	manager = &ManagerStruct{
		config: config,
		sessions: make(map[string]*Session),
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
		State: state,
		RedirectURI: authURI,
		Token: nil,
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

func (m *ManagerStruct) DeleteSession(userID string) {
	delete(m.sessions, userID)
}

func (m *ManagerStruct) SetToken(state *string, token *oauth2.Token) (error) {
	// O(n) instead of O(1) consider optimizing
	for _, session := range m.sessions {
		if session.State == *state {
			session.Token = token
			return nil
		}
	}
	return fmt.Errorf("couldn't find state")
}

// functions that make it easier to work with oauth
func (m *ManagerStruct) Exchange(ctx context.Context, code string, state string) (error) {
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