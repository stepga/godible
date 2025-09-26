package main

// test files https://mauvecloud.net/sounds/

import (
	"log"
	"os"

	. "github.com/stepga/godible/internal"
)

func main() {
	// TODO: wrap pin setup & pin-func-register-stuff up into one single func
	gpioSetupFailed := false
	gpioNames := []string{"GPIO4", "GPIO23", "GPIO24"}
	for _, gpioName := range gpioNames {
		pinIO, err := SetupPinByGPIOName(gpioName)
		if err != nil {
			log.Printf("godible: setup %s failed: %s", gpioName, err)
			gpioSetupFailed = true
			continue
		}

		err = GetPinCurrentFunction(pinIO)
		if err != nil {
			log.Printf("godible: gpio %s may not work, querying its function failed: %s", gpioName, err)
		}

		// TODO: register dedicated player (play/pause next/previous functions)
		// TODO: distinguish short vs long button press
		go CallFuncOnPinEdge(pinIO, func() {
			log.Printf("triggered %s\n", gpioName)
		})
	}
	if gpioSetupFailed {
		os.Exit(1)
	}

	// TODO: setup some sort of singleton player state instance
	//   - current song
	//   - offset/position
	//   - previous song
	//   - next song
	//   - state (pause/play)

	player, err := NewPlayer()
	if err != nil {
		log.Fatalf("godible: initializing player failed: %s", err)
	}
	defer player.Close()

	player.Play()

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
