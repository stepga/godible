package godible

import (
	"log"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	host "periph.io/x/host/v3"
)

type PinLevelMessage struct {
	State gpio.Level
	Reset gpio.Level
}

func SetupGPIOInput(pinName string, levelChan chan PinLevelMessage) (gpio.PinIO, error) {
	log.Printf("Loading periph.io drivers")
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	// Find Pin by name
	p := gpioreg.ByName(pinName)

	// Configure Pin for input, configure pull as needed
	// Edge mode is currently not supported
	if err := p.In(gpio.PullUp, gpio.NoEdge); err != nil {
		return nil, err
	}

	// Setup Input signalling
	go func() {
		lastLevel := p.Read()
		// How often to poll levels, 100-150ms is fairly responsive unless
		// button presses are very fast.
		// Shortening the polling interval <100ms significantly increases
		// CPU load.
		for range time.Tick(100 * time.Millisecond) {
			currentLevel := p.Read()
			log.Printf("level: %v", currentLevel)

			if currentLevel != lastLevel {
				levelChan <- PinLevelMessage{State: currentLevel, Reset: !currentLevel}
				lastLevel = currentLevel
			}
		}
	}()
	return p, nil
}
