package godible

import (
	"container/list"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestInitHttpHandlers(t *testing.T) {
	p := &Player{}
	tracklist := list.New()
	// FIXME: CreateTrackList takes forever ... TODO: speed up
	os.MkdirAll("/tmp/empty", 0o755)
	err := CreateTrackList(tracklist, "/tmp/empty")
	if err != nil {
		t.Fatalf("CreateTrackList failed: %+v", err)
	}
	p.TrackList = tracklist
	err = InitHttpHandlers(p)
	if err != nil {
		t.Fatalf("InitHttpHandlers failed: %+v", err)
	}

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
