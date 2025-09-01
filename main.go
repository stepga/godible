package main

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/pin"
	"periph.io/x/host/v3"
)

func main() {
	// Load all the drivers:
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Lookup a pin by its number:
	p := gpioreg.ByName("GPIO4")
	if p == nil {
		log.Fatal("Failed to find GPIO4")
	}

	//fmt.Printf("%s: %s\n", p, p.Function())
	pf, ok := p.(pin.PinFunc)
	if !ok {
		log.Fatal("pin.PinFunc is not implemented")
	}
	fmt.Printf("%s: %s\n", p, pf.Func())

	// Set it as input, with an internal pull down resistor:
	// XXX // if err := p.In(gpio.PullDown, gpio.BothEdges); err != nil {
	if err := p.In(gpio.PullDown, gpio.RisingEdge); err != nil {
		log.Fatal(err)
	}

	// Wait for edges as detected by the hardware, and print the value read:
	for {
		edgeDetected := p.WaitForEdge(0)
		if !edgeDetected {
			fmt.Printf("%s -> no edge detected (timeout); this should not happen, as we wait indefinitely\n", time.Now().Format("15:04:05"))
			continue
		}
		fmt.Printf("%s -> %s\n", time.Now().Format("15:04:05"), p.Read())
	}
}
