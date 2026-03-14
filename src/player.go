package godible

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/anisse/alsa"
)

type CommandVal int

const (
	TOGGLE CommandVal = iota
	NEXT
	PREVIOUS
)

const DATADIR = "/perm/godible-data/"

type Player struct {
	// commandMutex is needed to limit the concurrently executed commands
	// to one command
	commandMutex sync.Mutex
	// currentMutex is needed as both the Command functions as well as the
	// Play goroutine simultaneously access Player.current
	currentMutex sync.Mutex
	// TrackList represents the files located in DATADIR. Currently, it is
	// only created in NewPlayer and never updated.
	TrackList       *list.List
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc
	// current is currently played (or paused) Track
	current *list.Element
	// playSignal is used to signal Player to play the Player.current
	playSignal chan bool
	// playing represents Player's state of playing or pausing
	playing bool
	// maintain a mapping of RFID UIDs and Track
	rtm *RfidTrackManager
}

var cancelReasonNext = errors.New("next")
var cancelReasonPrevious = errors.New("previous")
var cancelReasonPause = errors.New("pause")

func NewPlayer() (*Player, error) {
	trackList := list.New()

	// XXX: NewTrack takes almost 1s for a 50mb MP3 file.
	//      For faster startup, create the tracklist in parallel.
	go func() {
		err := CreateTrackList(trackList, DATADIR)
		if err != nil {
			slog.Error("CreateTrackList failed", "err", err)
			os.Exit(1)
		}
	}()
	return &Player{
		TrackList:  trackList,
		current:    trackList.Front(),
		playSignal: make(chan bool),
		rtm:        newRfidTrackManager(),
	}, nil
}

func (player *Player) findTrackElement(track *Track) *list.Element {
	element := player.TrackList.Front()
	for element != nil {
		trackIterate, _ := element.Value.(*Track)
		if trackIterate == nil {
			slog.Error("findTrackElement: the Tracklist's element stored an invalid Track (this should not happen)")
			continue
		}
		if trackIterate == track {
			return element
		}
		element = element.Next()
	}
	return nil
}

func (player *Player) findTrack(trackPath string) *Track {
	element := player.TrackList.Front()
	for element != nil {
		track, _ := element.Value.(*Track)
		if track == nil {
			slog.Error("findTrack: the Tracklist's element stored an invalid Track (this should not happen)")
			continue
		}
		if track.Path == trackPath {
			return track
		}
		element = element.Next()
	}
	return nil
}

func (player *Player) getCurrent() *Track {
	var track *Track

	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		track, _ = player.current.Value.(*Track)
	}
	return track
}

func (player *Player) setCurrent(track *Track) {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	element := player.findTrackElement(track)
	if element == nil {
		slog.Error("setCurrent: failed to find Tracklist's element for track", "track", track)
		return
	}
	player.current = element
}

func (player *Player) setCurrentPrevious() {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		player.current = player.current.Prev()
	}
	if player.current == nil {
		player.current = player.TrackList.Back()
	}
}

func (player *Player) setCurrentNext() {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		player.current = player.current.Next()
	}
	if player.current == nil {
		player.current = player.TrackList.Front()
	}
}

func sampleRateSupported(sampleRate int) bool {
	switch sampleRate {
	case 44100:
		return true
	case 48000:
		return true
	default:
		return false
	}
}

func doPlay(ctx context.Context, t *Track) error {
	slog.Debug("doPlay begin", "Track", t.String())

	// XXX: keep bufferSizeInBytes to fixed 4kB for now
	bufferSizeInBytes := 4096
	alsaplayer, err := alsa.NewPlayer(
		t.metadata.sampleRate,
		2, // anisse/alsa: enforce two channels, even for mono files
		t.metadata.bytesPerSample,
		bufferSizeInBytes,
	)
	if err != nil {
		return err
	}
	defer alsaplayer.Close()

	reader, err := NewTrackReader(t)
	if err != nil {
		return err
	}
	defer reader.Close()

	if t.paused {
		_, err := reader.Seek(t.position, 0)
		if err != nil {
			return err
		}
		slog.Debug("continue paused track", "Track", t.String())
	}

	// alsaplayer.Write is not abortable/interruptable. WriteCtx is
	// interruptable by introducing a contexed and buffered write.
	err = WriteCtx(ctx, alsaplayer, reader, t)
	if err == context.Canceled && context.Cause(ctx) == cancelReasonPause {
		t.paused = true
	} else {
		t.paused = false
	}
	return err
}

func (player *Player) Play() {
	for {
		<-player.playSignal

		for {
			t := player.getCurrent()
			if t == nil {
				// TODO: would be nice to forward/tee error messages into the webgui
				slog.Error("could not fetch current Track, wait & try again")
				time.Sleep(100 * time.Millisecond)
				player.current = player.TrackList.Front()
				continue
			}

			player.playing = true
			err := doPlay(player.ctx, t)
			player.playing = false

			if err == context.Canceled {
				slog.Debug("interrupt/cancelation", "Track", t.String())
				break
			} else if err != nil {
				slog.Error("doPlay() failed", "Track", t.String(), "error", err)
			}
			player.setCurrentNext()
		}
	}
}

func (player *Player) sendPlaySignal() {
	for attempt := range 10 {
		select {
		case player.playSignal <- true:
			return
		default:
			slog.Debug("missing receiver sent signal on playSignal", "attempt", attempt)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (player *Player) resetCancel(cancelReason error) {
	if player.cancelCauseFunc != nil {
		player.cancelCauseFunc(cancelReason)
	}
	ctx, cancelfunc := context.WithCancelCause(context.Background())
	player.ctx = ctx
	player.cancelCauseFunc = cancelfunc
}

func (player *Player) doToggle() {
	wasPlaying := player.playing
	player.resetCancel(cancelReasonPause)
	if !wasPlaying {
		player.sendPlaySignal()
	}
}

func (player *Player) doNext() {
	player.resetCancel(cancelReasonNext)
	player.setCurrentNext()
	player.sendPlaySignal()
}

func (player *Player) doPrevious() {
	player.resetCancel(cancelReasonPrevious)
	player.setCurrentPrevious()
	player.sendPlaySignal()
}

func (player *Player) Command(cmd CommandVal) {
	player.commandMutex.Lock()
	defer player.commandMutex.Unlock()

	switch cmd {
	case NEXT:
		player.doNext()
	case PREVIOUS:
		player.doPrevious()
	case TOGGLE:
		player.doToggle()
	default:
		slog.Error("unknown command", "cmd", cmd)
	}
}

func (player *Player) RfidUidReceiver(uidpass chan string) {
	go func() {
		for {
			slog.Info("RfidUidReceiver: wait for new RFID UID")
			uid := <-uidpass

			if player.rtm.SetMapping(uid) == true {
				slog.Info("linked RFID UID to current TrackTrainer", "uid", uid)
				continue
			} else {
				slog.Debug("no rfid-track-linking to learn")
			}

			// TODO: ignore uid if learning happend the last 3 seconds

			track := player.rtm.GetTrack(uid)
			if track == nil {
				slog.Error("could not find track for given rfid uid", "uid", uid)
				continue
			}
			if track == player.getCurrent() {
				slog.Debug("respective track already playing, do nothing", "uid", uid)
				continue
			}

			slog.Debug("about to play track corresponding to rfid uid", "uid", uid, "track", track.String())
			// pause currently played track; will save the current position
			if player.playing {
				player.Command(TOGGLE)
				time.Sleep(50 * time.Millisecond)
			}
			// play the new track
			player.setCurrent(track)
			player.Command(TOGGLE)
		}
	}()
}
