package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/oauth2"
)

var config = &oauth2.Config{
	ClientID: os.Getenv("ION_CLIENT_ID"),
	ClientSecret: os.Getenv("ION_SECRET_ID"),
	Scopes: []string{"read", "write"},
	Endpoint: oauth2.Endpoint{
		TokenURL: os.Getenv("OAUTH_TOKEN_URL"),
		AuthURL: os.Getenv("OAUTH_CODE_URL"),
	},
}

type Session struct {
	State string;
	RedirectURI string;
	Token *oauth2.Token;
}

type ManagerStruct struct {
	config *oauth2.Config;
	sessions map[int]*Session // <state>: <session_struct>
}

var manager* ManagerStruct // store a file-wide manager

func init() {
	manager = &ManagerStruct{
		config: config,
		sessions: make(map[int]*Session),
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

func (m *ManagerStruct) CreateNewSession(userID int) (*Session, error) {
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

func (m *ManagerStruct) GetSession(userID int) (*Session, error) {
	session, ok := m.sessions[userID]

	if !ok {
		return nil, fmt.Errorf("couldn't get session")
	}

	return session, nil
}

func (m *ManagerStruct) DeleteSession(userID int) {
	delete(m.sessions, userID)
}