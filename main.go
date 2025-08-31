package main

import (
	"log"

	godible "github.com/stepga/godible/core"
)

func main() {
	// Channel for communicating Pin levels
	levelChan := make(chan godible.PinLevelMessage)

	p, err := godible.SetupGPIOInput("GPIO24", levelChan)
	if err != nil {
		log.Fatal(err)
	}

	// Main loop, act on level changes
	for {
		select {
		case msg := <-levelChan:
			if msg.State {
				log.Printf("Pin %s is High, processing high state tasks", p.Name())
				// Process high state tasks
			} else if msg.Reset {
				log.Printf("Pin %s is Low, resetting to wait for high state", p.Name())
				// Process resetting logic, if any
			}
		default:
			// Any other ongoing tasks
		}
	}
}
