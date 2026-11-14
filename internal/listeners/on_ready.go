package listeners

import (
	"log/slog"
	"sync"
	"time"

	"github.com/deR0R0/isotope-go/internal/commands"
	"github.com/disgoorg/disgo/events"
)

var tickerSync sync.Once

func OnReady(event *events.Ready) {
	slog.Info("logged in as user " + event.User.Username)
	slog.Info("bot is ready to recieve commands")
	// register our permannet button
	commands.RegisterButton("/button/isotope_authorize", commands.LoginButtonHandler)
	// create the "tasks" loop
	tickerSync.Do(func() {
		go func() {
			slog.Info("started goroutine for cleaning up component lookup")
			ticker := time.NewTicker(2 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				commands.CleanupExpiredComponents()
			}
		}()
	})
}
