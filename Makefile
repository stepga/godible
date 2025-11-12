.PHONY: all rpi tar

all: rpi

# quick test: cross compile the binary and scp it (via breakglass) onto the raspberry pi
rpi:
	 env GOOS=linux GOARCH=arm64 go build

tar:
	rm -f _gokrazy/*.tar
	tar cf _gokrazy/extrafiles_arm64.tar -C _gokrazy etc
