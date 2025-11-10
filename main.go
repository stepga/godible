package main

// test files https://mauvecloud.net/sounds/

import (
	"log/slog"
	"os"

	. "github.com/stepga/godible/src"
)

func main() {
	SetDefaultLogger(slog.LevelDebug)

	player, err := NewPlayer()
	if err != nil {
		slog.Error("NewPlayer: initializing player failed", "err", err)
		os.Exit(1)
	}

	err = RegisterPinFunc(
		"GPIO4",
		func() {
			player.Command(PREVIOUS)
		},
		func() {
			slog.Info("TODO: implement long previous button")
		},
	)
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}

	err = RegisterPinFunc(
		"GPIO23",
		func() {
			player.Command(TOGGLE)
		},
		func() {
			slog.Info("rebooting device")
			Reboot()
		},
	)

	err = RegisterPinFunc(
		"GPIO24",
		func() {
			player.Command(NEXT)
		},
		func() {
			slog.Info("TODO: implement long next button")
		},
	)
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}

	err = InitHttpHandlers(player)
	if err != nil {
		slog.Error("InitHttpHandlers failed", "err", err)
		os.Exit(1)
	}

	player.Play()
}
