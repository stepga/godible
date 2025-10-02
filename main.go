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
		slog.Debug("GPIO4: trigger PREVIOUS")
		player.Command <- CMD_PREVIOUS
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	err = RegisterPinFunc("GPIO23", func() {
		slog.Debug("GPIO23: trigger TOGGLE")
		player.Command <- CMD_TOGGLE
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	err = RegisterPinFunc("GPIO24", func() {
		slog.Debug("GPIO24: trigger NEXT")
		player.Command <- CMD_NEXT
	})
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}
	go player.Run()

	// TODO: web interface
	//   - upload songs
	//     - plain mp3/wav files
	//     - directory with files
	//   - restructure files/directories
	//   - spotify (https://github.com/anisse/librespot-golang)

	// TODO: usb webcam module && qr code recognition

	// block main goroutine forever
	<-make(chan struct{})
}
