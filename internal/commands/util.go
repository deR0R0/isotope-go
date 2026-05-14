package commands

import (
	"log/slog"
	"time"
)

func DeleteAfter(delay time.Duration, deleteFunction func() error) {
	go func() {
		time.Sleep(delay)
		if err := deleteFunction(); err != nil {
			slog.Error("err while deleting message, possibly safe to ignore", slog.String("err", err.Error()))
		}
	}()
}

func ShowErrorMessage(source string, editFunction func() error) {
	slog.Error(source + " had an error. giving the user response message")
	if err := editFunction(); err != nil {
		slog.Error("wow, another err while editing the function.")
	}
}
