package webserver

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/deR0R0/isotope-go/internal/commands"
	"github.com/deR0R0/isotope-go/internal/db"
	"github.com/deR0R0/isotope-go/internal/oauth"
	"golang.org/x/oauth2"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "isotope, home of better code because i'm slightly smarter. wait. am i??")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		fmt.Fprintf(w, "Missing code and state for request.")
		return
	}

	// exchange the code for a token
	httpClient := &http.Client{Timeout: 2 * time.Second}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	err := oauth.Manager().Exchange(ctx, code, state)
	if err != nil {
		fmt.Fprintf(w, "%s", "There was an error while exchanging for your token! Try again later. DM an admin with this err: "+err.Error())
		return
	}

	// actually get the bot to add the roles to the user
	err = commands.HandleNewLogin(state)
	if err != nil {
		fmt.Fprintf(w, "%s", "Oops, there was an issue while adding your discord roles. DM an admin with this err: "+err.Error())
		return
	}

	userid, _ := db.GetUserFromState(db.GetDB(), state)
	profile, err := oauth.Manager().GetProfile(userid)
	if err != nil {
		slog.Error("error while getting profile", slog.String("err", err.Error()))
		fmt.Fprintf(w, "%s", "Oops, looks like there was an issue while fetching your profile... DM an admin with this err: "+err.Error())
		return
	}
	slog.Info("testing user's oauth session by sending a profile request...")

	fmt.Fprintf(w, "%s", "<h1>Welcome, "+profile.Ion_Username+"</h1>\n<p>Hello "+profile.Full_Name+"! You may return to discord.</p><p>Details: "+profile.Ion_Username+" is linked to "+userid)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok"}`)
}

func Run() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/health", healthHandler)

	portString := os.Getenv("WEB_SERVER_PORT")
	port, err := strconv.Atoi(portString)

	if err != nil {
		slog.Error("couldn't start up the server beacuse the web server port is probs not a number.")
		return
	}

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server listening on http://localhost%s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
