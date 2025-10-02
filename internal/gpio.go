package godible

import (
	"fmt"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/pin"
	host "periph.io/x/host/v3"
)

type pinfunction func()

func GetPinCurrentFunction(pinIO gpio.PinIO) error {
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

// InitHostDrivers initialises all the relevant host drivers.
//
// It is safe to call this function multiple times, as the underlying function
// saves the previous returned state on later calls.
func InitHostDrivers() error {
	_, err := host.Init()
	return err
}

func SetupPinByGPIOName(gpioName string) (gpio.PinIO, error) {
	if err := InitHostDrivers(); err != nil {
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

func CallFuncOnPinEdge(pinIO gpio.PinIO, fn pinfunction) {
	for {
		// TODO: also possible to detect long press
		// - save pre state
		// - on edge:
		//   - loop (couple of ms sleep)
		//   - if critical time achieved: trigger long press func (additional parameter)
		//   - otherwise short press func
		edgeDetected := pinIO.WaitForEdge(0)
		if !edgeDetected {
			fmt.Println("gpio (PinEdgeCallback): this should not have happen ...")
			continue
		}
		fn()
	}
}
