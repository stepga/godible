package main

// test files https://mauvecloud.net/sounds/

import (
	"log/slog"
	"os"

	. "github.com/stepga/godible/internal"
)

func main() {

	// TODO: setup some sort of singleton player state instance
	//   - current song
	//   - offset/position
	//   - previous song
	//   - next song
	//   - state (pause/play)

	SetDefaultLogger(slog.LevelDebug)

	player, err := NewPlayer()
	if err != nil {
		slog.Error("NewPlayer: initializing player failed", "err", err)
		os.Exit(1)
	}
	go player.Run()

	// TODO: wrap pin setup & pin-func-register-stuff up into one single func
	gpioSetupFailed := false
	gpioNames := []string{"GPIO4", "GPIO23", "GPIO24"}
	for _, gpioName := range gpioNames {
		pinIO, err := SetupPinByGPIOName(gpioName)
		if err != nil {
			slog.Error("SetupPinByGPIOName failed", "gpioName", gpioName, "err", err)
			gpioSetupFailed = true
			continue
		}

		err = GetPinCurrentFunction(pinIO)
		if err != nil {
			slog.Error("GetPinCurrentFunction failed, respective gpio may not work", "gpioName", gpioName, "err", err)
		}

		// TODO: register dedicated player (play/pause next/previous functions)
		// TODO: distinguish short vs long button press
		go CallFuncOnPinEdge(pinIO, func() {
			slog.Debug("CallFuncOnPinEdge triggered", "gpioName", gpioName)
			player.Stop()
		})
	}
	if gpioSetupFailed {
		os.Exit(1)
	}

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
