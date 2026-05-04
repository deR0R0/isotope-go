package oauth

import (
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

