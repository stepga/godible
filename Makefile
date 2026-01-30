.PHONY: all amd64 arm64

all: arm64

# raspberry pi
arm64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build

amd64:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
