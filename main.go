package main

// test files https://mauvecloud.net/sounds/

import (
	"log/slog"
	"os"

	. "github.com/stepga/godible/internal"
)

func main() {
	SetDefaultLogger(slog.LevelDebug)

	player, err := NewPlayer()
	if err != nil {
		slog.Error("NewPlayer: initializing player failed", "err", err)
		os.Exit(1)
	}

	err = RegisterPinFunc("GPIO4", func() {
		player.Command(PREVIOUS)
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	err = RegisterPinFunc("GPIO23", func() {
		player.Command(TOGGLE)
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	err = RegisterPinFunc("GPIO24", func() {
		player.Command(NEXT)
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	go player.Play()

	// block main goroutine forever
	<-make(chan struct{})
}
