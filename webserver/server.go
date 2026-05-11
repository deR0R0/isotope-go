package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/deR0R0/isotope-go/internal/oauth"
	"golang.org/x/oauth2"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World! The time is %s\n", time.Now().Format(time.RFC1123))
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
		fmt.Fprintf(w, "There was an error while logging you in! Try again later. DM @robboach")
	}

	fmt.Fprintf(w, "Successfully Connected. You may return to Discord. It may take a while for it to update.")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok"}`)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/health", healthHandler)

	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Server listening on http://localhost%s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
