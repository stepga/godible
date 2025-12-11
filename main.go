package main

// test files https://mauvecloud.net/sounds/

import (
	"log/slog"
	"os"

	"encoding/hex"
	"fmt"
	"time"

	. "github.com/stepga/godible/src"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/mfrc522"
	"periph.io/x/host/v3"
)

// mfrc522 rfid device
var rfid *mfrc522.Dev

// spi port
var port spi.PortCloser

// pins used for rest and irq
const (
	resetPin = "P1_22" // GPIO 25
	irqPin   = "P1_12" // GPIO 18
)

/*
Setup inits and starts hardware.
*/
func setup() {
	var err error

	// guarantees all drivers are loaded.
	if _, err = host.Init(); err != nil {
		slog.Error("host.Init failed", "err", err)
		os.Exit(1)
	}

	// get the first available spi port eith empty string.
	for {
		time.Sleep(500 * time.Millisecond)
		port, err = spireg.Open("")
		if err != nil {
			slog.Error("spireg.Open failed", "err", err)
			//os.Exit(1)
		}
		if err == nil {
			break
		}
	}

	// get GPIO rest pin from its name
	var gpioResetPin gpio.PinOut = gpioreg.ByName(resetPin)
	if gpioResetPin == nil {
		slog.Error("gpioreg.ByName failed", "err", err)
		os.Exit(1)
	}

	// get GPIO irq pin from its name
	var gpioIRQPin gpio.PinIn = gpioreg.ByName(irqPin)
	if gpioIRQPin == nil {
		slog.Error("gpioreg.ByName failed", "err", err)
		os.Exit(1)
	}

	rfid, err = mfrc522.NewSPI(port, gpioResetPin, gpioIRQPin, mfrc522.WithSync())
	if err != nil {
		slog.Error("mfrc522.NewSPI failed", "err", err)
		os.Exit(1)
	}

	// setting the antenna signal strength, signal strength from 0 to 7
	rfid.SetAntennaGain(5)

	fmt.Println("Started rfid reader.")
}

// close is idling the RFID device and closes spi port.
func close() {

	if err := rfid.Halt(); err != nil {
		slog.Error("rfid.Halt failed", "err", err)
		os.Exit(1)
	}

	if err := port.Close(); err != nil {
		slog.Error("port.Close", "err", err)
		os.Exit(1)
	}

}

// stringIntoByte16 converst the given str into 16 bytes.
// String that are longer than 16 bytes, will be cut.
func stringIntoByte16(str string) [16]byte {
	var data [16]byte
	copy(data[:], str) // copy already checks length of str
	return data
}

// find first null byte
func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

func main() {
	SetDefaultLogger(slog.LevelDebug)

	// init rfid hardware
	setup()

	// read rfid UID
	for {
		data, err := rfid.ReadUID(5 * time.Second)
		if err != nil {
			slog.Error("rfid.ReadUID failed", "err", err)
			os.Exit(1)
		} else {
			slog.Debug("rfid.ReadUID read data", "data", hex.EncodeToString(data))
		}
		if int(data[0]) == 10 {
			break //unreachable
		}
		time.Sleep(time.Millisecond * 500)
	}

	player, err := NewPlayer()
	if err != nil {
		slog.Error("NewPlayer: initializing player failed", "err", err)
		os.Exit(1)
	}

	err = RegisterPinFunc(
		"GPIO4",
		func() {
			player.Command(PREVIOUS)
		},
		func() {
			slog.Info("TODO: implement long previous button")
		},
	)
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}

	err = RegisterPinFunc(
		"GPIO23",
		func() {
			player.Command(TOGGLE)
		},
		func() {
			slog.Info("rebooting device")
			Reboot()
		},
	)

	err = RegisterPinFunc(
		"GPIO24",
		func() {
			player.Command(NEXT)
		},
		func() {
			slog.Info("TODO: implement long next button")
		},
	)
	if err != nil {
		slog.Error("RegisterPinFunc failed", "err", err)
		os.Exit(1)
	}

	err = InitHttpHandlers(player)
	if err != nil {
		slog.Error("InitHttpHandlers failed", "err", err)
		os.Exit(1)
	}

	player.Play()
}
