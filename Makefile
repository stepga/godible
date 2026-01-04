.PHONY: all amd64 arm64

all: arm64

# raspberry pi
arm64:
	 env GOOS=linux GOARCH=arm64 go build

amd64:
	 env GOOS=linux GOARCH=amd64 go build
