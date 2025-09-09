package main

// test files https://mauvecloud.net/sounds/

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/anisse/alsa"
	godible_core "github.com/stepga/godible/core"
)

func main() {
	gpioSetupFailed := false
	gpioNames := []string{"GPIO4", "GPIO23", "GPIO24"}
	for _, gpioName := range gpioNames {
		pinIO, err := godible_core.SetupPinByGPIOName(gpioName)
		if err != nil {
			log.Printf("godible: setup %s failed: %s", gpioName, err)
			gpioSetupFailed = true
			continue
		}

		err = godible_core.PinCurrentFunction(pinIO)
		if err != nil {
			log.Printf("godible: gpio %s may not work, querying its function failed: %s", gpioName, err)
		}

		// TODO: register dedicated player (play/pause next/previous functions)
		// TODO: distinguish short vs long button press
		go godible_core.PinEdgeCallback(pinIO, func() {
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

	// TODO: web interface
	//   - upload songs
	//     - plain mp3/wav files
	//     - directory with files
	//   - restructure files/directories
	//   - spotify (https://github.com/anisse/librespot-golang)

	// TODO: usb webcam module && qr code recognition

	p, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		panic(err.Error())
	}
	defer p.Close()

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		panic(err.Error())
	}
	_, err = p.Write(b)
	if err != nil {
		panic(err.Error())
	}

	for {
		time.Sleep(5 * time.Second)
	}
}
