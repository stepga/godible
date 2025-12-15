package godible

import (
	"log/slog"
	"strings"

	"encoding/hex"
	"fmt"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/mfrc522"
)

const (
	resetPin        = "P1_22" // GPIO 25
	irqPin          = "P1_12" // GPIO 18
	uidWaitDuration = 5 * time.Second
)

type RfidDevice struct {
	device        *mfrc522.Dev
	spiPortCloser spi.PortCloser
}

// Soft-stop the RFID chip and close the spi port.
func (rfid *RfidDevice) Close() {
	if rfid.device != nil {
		err := rfid.device.Halt()
		if err != nil {
			slog.Error("rfid.Halt failed", "err", err)
		}
	}
	if rfid.spiPortCloser != nil {
		err := rfid.spiPortCloser.Close()
		if err != nil {
			slog.Error("spiPortCloser.Close", "err", err)
		}
	}
}

func NewRfidDevice() (*RfidDevice, error) {
	var err error
	ret := RfidDevice{}

	if err = initHostDrivers(); err != nil {
		return nil, err
	}

	// get the first available spi port (usually SPI0.0)
	// XXX: with extra bootloader "spi channel select setting",
	//      e.g.: "dtoverlay=spi1-1cs,cs0_pin=12",
	//      the bus changes to "SPI1.0" (which is better passed this explicitly)
	ret.spiPortCloser, err = spireg.Open("")
	if err != nil {
		return nil, fmt.Errorf("spireg.Open: %w", err)
	}

	// get GPIO rest pin from its name
	var gpioResetPin gpio.PinOut = gpioreg.ByName(resetPin)
	if gpioResetPin == nil {
		ret.Close()
		return nil, fmt.Errorf("gpioreg.ByName: %w", err)
	}

	// get GPIO irq pin from its name
	var gpioIRQPin gpio.PinIn = gpioreg.ByName(irqPin)
	if gpioIRQPin == nil {
		ret.Close()
		return nil, fmt.Errorf("gpioreg.ByName: %w", err)
	}

	ret.device, err = mfrc522.NewSPI(ret.spiPortCloser, gpioResetPin, gpioIRQPin, mfrc522.WithSync())
	if err != nil {
		ret.Close()
		return nil, fmt.Errorf("mfrc522.NewSPI: %w", err)
	}

	// setting the antenna signal strength, signal strength from 0 to 7
	ret.device.SetAntennaGain(7)

	return &ret, nil
}

func (rfid *RfidDevice) ReadUID(duration time.Duration) (string, error) {
	data, err := rfid.device.ReadUID(duration)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		slog.Info("ReadDir returned no data but did no fail")
	}
	return hex.EncodeToString(data), nil
}

func errIsRfidTimeout(err error) bool {
	return strings.HasPrefix(
		err.Error(),
		"mfrc522 lowlevel: timeout waiting for IRQ",
	)
}

func (rfid *RfidDevice) RfidUidWorker(uidPass chan string) {
	failCounterMax := 10
	failCounter := 0
	go func() {
		for {
			time.Sleep(1 * time.Second)
			ret, err := rfid.ReadUID(uidWaitDuration)
			if err != nil && !errIsRfidTimeout(err) {
				slog.Error("rfid.Read failed", "err", err)
				failCounter = failCounter + 1
				if failCounter > failCounterMax {
					slog.Error("rfid.Read reached maximum amount of errors: abort")
					break
				}
			} else {
				failCounter = 0
				if len(ret) == 0 {
					slog.Debug("skip passing empty rfid uid")
					continue
				}
				slog.Debug("pass rfid uid into channel", "uid", ret)
				uidPass <- ret
				slog.Debug("successfully passed rfid uid", "uid", ret)
			}
		}
	}()
}
