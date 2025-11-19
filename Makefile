.PHONY: all rpi tar

all: rpi

# quick test: cross compile the binary and scp it (via breakglass) onto the raspberry pi
rpi:
	 env GOOS=linux GOARCH=arm64 go build
