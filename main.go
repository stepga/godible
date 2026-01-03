package main

import (
	"log/slog"
	"os"

	. "github.com/stepga/godible/src"
)

func main() {
	SetDefaultLogger(slog.LevelDebug)

	slog.Info("remount /perm to read-only initially")
	err := RemountPerm(true)
	if err != nil {
		slog.Error("RemountPerm failed", "err", err)
	}

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

	rfid, err := NewRfidDevice()
	uidPassChan := make(chan string)
	rfid.RfidUidSender(uidPassChan)
	player.RfidUidReceiver(uidPassChan)

	player.Play()
}
