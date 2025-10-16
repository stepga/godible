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
	"github.com/go-audio/wav"
)

const (
	DATADIR      = "/perm/godible-data/"
	CMD_TOGGLE   = "TOGGLE"
	CMD_NEXT     = "NEXT"
	CMD_PREVIOUS = "PREVIOUS"
)

type CommandVal int

const (
	TOGGLE CommandVal = iota
	NEXT
	PREVIOUS
)

type Player struct {
	// commandMutex is needed to limit the concurrently executed commands
	// to one command
	commandMutex sync.Mutex
	// currentMutex is needed as both the Command functions as well as the
	// Play goroutine simultaneously access Player.current
	currentMutex sync.Mutex
	// trackList represents the files located in DATADIR. Currently, it is
	// only created in NewPlayer and never updated.
	trackList       *list.List
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc
	// current is currently played (or paused) Track
	current *list.Element
	// playSignal is used to signal Player to play the Player.current
	playSignal chan bool
	// playing represents Player's state of playing or pausing
	playing bool
}

// a WAV file's metadata used for initializing alsa.NewPlayer
type WavMetadata struct {
	bytesPerSample int
	sampleRate     int
	channelNum     int
}

var cancelReasonNext = errors.New("next")
var cancelReasonPrevious = errors.New("previous")
var cancelReasonPause = errors.New("pause")

func newWavMetadata(path string) (*WavMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d := wav.NewDecoder(f)
	d.ReadInfo()
	err = d.Err()
	if err != nil {
		return nil, err
	}
	return &WavMetadata{
		bytesPerSample: int(d.SampleBitDepth() / 8),
		sampleRate:     int(d.SampleRate),
		channelNum:     int(d.NumChans),
	}, nil
}

func NewPlayer() (*Player, error) {
	trackList := list.New()
	err := CreateTrackList(trackList, DATADIR)
	if err != nil {
		return nil, err
	}
	slog.Debug("gathered files", "len", trackList.Len())
	return &Player{
		trackList:  trackList,
		current:    trackList.Front(),
		playSignal: make(chan bool),
	}, nil
}

func (player *Player) getCurrent() *Track {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		t, ok := player.current.Value.(*Track)
		if ok {
			return t
		}
	}
	return nil
}

func (player *Player) setCurrentPrevious() {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		player.current = player.current.Prev()
	}
	if player.current == nil {
		player.current = player.trackList.Back()
	}
}

func (player *Player) setCurrentNext() {
	player.currentMutex.Lock()
	defer player.currentMutex.Unlock()

	if player.current != nil {
		player.current = player.current.Next()
	}
	if player.current == nil {
		player.current = player.trackList.Front()
	}
}

func doPlay(ctx context.Context, t *Track) error {
	slog.Debug("doPlay begin", "Track", t.String())

	wavMetadata, err := newWavMetadata(t.path)
	if err != nil {
		return err
	}
	// XXX: alsaplayer discards mono wav files, but it seems to work anyways
	channelNum := max(wavMetadata.channelNum, 2)
	// XXX: keep bufferSizeInBytes to fixed 4kB for now
	bufferSizeInBytes := 4096
	alsaplayer, err := alsa.NewPlayer(
		wavMetadata.sampleRate,
		channelNum,
		wavMetadata.bytesPerSample,
		bufferSizeInBytes,
	)
	if err != nil {
		return err
	}
	defer alsaplayer.Close()

	file, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer file.Close()

	if t.offset != 0 {
		_, err := file.Seek(t.offset, 0)
		if err != nil {
			return err
		}
		slog.Debug("continue paused title", "Track", t.String())
	}

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	if err == context.Canceled && context.Cause(ctx) == cancelReasonPause {
		t.offset = t.offset + written_bytes
	} else {
		t.offset = 0
	}
	return err
}

func (player *Player) Play() {
	for {
		<-player.playSignal

		for {
			t := player.getCurrent()
			if t == nil {
				slog.Error("could not fetch current Track")
				os.Exit(1)
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
