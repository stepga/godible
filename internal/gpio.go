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

func callFuncOnPinEdge(pinIO gpio.PinIO, fnShort pinfunction, fnLong pinfunction) {
	for {
		edgeDetected := pinIO.WaitForEdge(0)
		if !edgeDetected {
			slog.Error("this should not have happen ...")
			continue
		}

		pressed_milliseconds := 0
		for {
			time.Sleep(5 * time.Millisecond)
			pressed_milliseconds = pressed_milliseconds + 5
			if !pinIO.Read() {
				break
			}
			if pressed_milliseconds > 1500 {
				fnLong()
				return
			}
		}
		slog.Debug("button pushed time", "milliseconds", pressed_milliseconds)

		fnShort()
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

	go callFuncOnPinEdge(
		pinIO,
		func() {
			slog.Debug("call pinfunction (short)", "gpioName", gpioName)
			fnShort()
		},
		func() {
			slog.Debug("call pinfunction (long)", "gpioName", gpioName)
			fnLong()
		},
	)
	return nil
}
