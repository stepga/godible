package main

import (
	"log"
	"os"
	"time"

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

		go godible_core.PinEdgeCallback(pinIO, func() {
			log.Printf("triggered %s\n", gpioName)
		})
	}
	if gpioSetupFailed {
		os.Exit(1)
	}
	for {
		time.Sleep(5 * time.Second)
	}
}
