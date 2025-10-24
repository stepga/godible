package godible

import (
	"fmt"
	"log/slog"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/pin"
	host "periph.io/x/host/v3"
)

type pinfunction func()

// initHostDrivers initialises all the relevant host drivers.
//
// It is safe to call this function multiple times, as the underlying function
// saves the previous returned state on later calls.
func initHostDrivers() error {
	_, err := host.Init()
	return err
}

const (
	TICK_PERIOD                = time.Millisecond * 15
	LONG_BUTTON_PRESS_DURATION = time.Millisecond * 1500
)

func getPinCurrentFunction(pinIO gpio.PinIO) error {
	if pinIO == nil {
		return fmt.Errorf("gpio: invalid argument (PinFunction): nil")
	}
	pinIOFunc, ok := pinIO.(pin.PinFunc)
	if !ok {
		return fmt.Errorf("gpio: pin '%s' does not have a function", pinIO.Name())
	}
	pinIOFunc.Func()
	return nil
}

func setupPinByGPIOName(gpioName string) (gpio.PinIO, error) {
	if err := initHostDrivers(); err != nil {
		return nil, err
	}

	pinIO := gpioreg.ByName(gpioName)
	if pinIO == nil {
		return nil, fmt.Errorf("gpio: GPIO pin for '%s' not found", gpioName)
	}

	err := pinIO.In(gpio.PullDown, gpio.RisingEdge)
	if err != nil {
		return nil, err
	}

	return pinIO, nil
}

func callFuncOnPinEdgeAndPoll(pinIO gpio.PinIO, fnShort pinfunction, fnLong pinfunction) {
	for {
		edgeDetected := pinIO.WaitForEdge(0)
		if !edgeDetected {
			slog.Error("this should not have happen ...")
			continue
		}

		// XXX: `deref ticker.Stop()` not needed:
		// [...] As of Go 1.23, the garbage collector can recover
		// unreferenced tickers even if they haven't been stopped.
		ticker := time.NewTicker(TICK_PERIOD)

		counter := 1
		long_triggered := false
		for range ticker.C {
			if pinIO.Read() {
				if long_triggered {
					continue
				}
				counter = counter + 1
				if counter*int(TICK_PERIOD) > int(LONG_BUTTON_PRESS_DURATION) {
					slog.Debug("trigger long pinfunction")
					fnLong()
					long_triggered = true
				}
			} else {
				if !long_triggered {
					slog.Debug("trigger short pinfunction")
					fnShort()
				}
				break
			}
		}
	}
}

func RegisterPinFunc(gpioName string, fnShort pinfunction, fnLong pinfunction) error {
	pinIO, err := setupPinByGPIOName(gpioName)
	if err != nil {
		return err
	}

	err = getPinCurrentFunction(pinIO)
	if err != nil {
		return fmt.Errorf("gpio: could not gather current function for pin '%s'", pinIO.Name())
	}

	go callFuncOnPinEdgeAndPoll(pinIO, fnShort, fnLong)
	return nil
}
