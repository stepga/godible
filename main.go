package main

import (
	"log"

	godible "github.com/stepga/godible/core"
)

func main() {
	pinLevelChannel := make(chan godible.PinLevelMessage)

	pin, err := godible.SetupGPIOInput("GPIO24", pinLevelChannel)
	if err != nil {
		log.Fatal(err)
	}

	// Main loop, act on level changes
	for pinLevelState := range pinLevelChannel {
		if pinLevelState.State {
			log.Printf("Pin %s is High", pin.Name())
		} else if pinLevelState.Reset {
			log.Printf("Pin %s is Low", pin.Name())
		} else {
			log.Fatalf("Unexpected pin state: %+v", pinLevelChannel)
		}
	}
}
