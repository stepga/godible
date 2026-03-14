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
	resetPin             = "P1_22" // GPIO 25
	irqPinName           = "P1_12" // GPIO 18
	uidWaitDuration      = 5 * time.Second
	readUIDSenderMaxFail = 20
)

type RfidDevice struct {
	*mfrc522.Dev
	spiPortCloser spi.PortCloser
}

// Soft-stop the RFID chip and close the spi port.
func (rfid *RfidDevice) Close() {
	if rfid != nil {
		err := rfid.Halt()
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
	err := initHostDrivers()
	if err != nil {
		return nil, err
	}

	// get the first available spi port (default "SPI0.0")
	// XXX: with extra bootloader "spi channel select setting",
	//      e.g.: "dtoverlay=spi1-1cs,cs0_pin=12",
	//      the bus changes to "SPI1.0" (which is better passed explicitly)
	spiPort, err := spireg.Open("")
	if err != nil {
		return nil, fmt.Errorf("spireg.Open: %w", err)
	}

	var gpioResetPin gpio.PinOut = gpioreg.ByName(resetPin)
	if gpioResetPin == nil {
		spiPort.Close()
		return nil, fmt.Errorf("gpioreg.ByName: %w", err)
	}

	var gpioIRQPin gpio.PinIn = gpioreg.ByName(irqPinName)
	if gpioIRQPin == nil {
		spiPort.Close()
		return nil, fmt.Errorf("gpioreg.ByName: %w", err)
	}

	rfidSpiDevice, err := mfrc522.NewSPI(spiPort, gpioResetPin, gpioIRQPin, mfrc522.WithSync())
	if err != nil {
		spiPort.Close()
		return nil, fmt.Errorf("mfrc522.NewSPI: %w", err)
	}
	rfidSpiDevice.SetAntennaGain(7)

	return &RfidDevice{rfidSpiDevice, spiPort}, nil
}

func (rfid *RfidDevice) ReadUIDString(duration time.Duration) (string, error) {
	data, err := rfid.ReadUID(duration)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		slog.Info("ReadUID returned no data but did no fail")
	}
	return hex.EncodeToString(data), nil
}

func errIsRfidTimeout(err error) bool {
	return strings.HasPrefix(
		err.Error(),
		"mfrc522 lowlevel: timeout waiting for IRQ",
	)
}

// RfidUidSender continuously reads RFID UIDs and passes them into its
// channel `uidPass`. On more than `readUIDSenderMaxFail` consecutive errors,
// the goroutine will return.
func (rfid *RfidDevice) RfidUidSender(uidPass chan string) {
	failCounter := 0
	go func() {
		for {
			time.Sleep(1 * time.Second)
			ret, err := rfid.ReadUIDString(uidWaitDuration)
			if err == nil {
				failCounter = 0
				if len(ret) != 0 {
					slog.Info("RfidUidSender rfid.ReadUIDString)", "uid", ret)
					uidPass <- ret
				}
				continue
			}
			if !errIsRfidTimeout(err) {
				failCounter = failCounter + 1
				slog.Error("rfid.ReadUIDString failed", "err", err, "failCounter", failCounter)
				if failCounter > readUIDSenderMaxFail {
					slog.Error("rfid.ReadUIDString reached maximum amount of errors: abort")
					return
				}
			}
		}
	}()
}
