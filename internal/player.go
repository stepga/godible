package godible

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/anisse/alsa"
	"github.com/go-audio/wav"
)

// TODO: instead of hardcoding nonsense strings introduce something like
//
//   type command int
//   const (
//     TOGGLE command = iota
//     NEXT
//     ...
//   )

const (
	DATADIR      = "/perm/godible-data/"
	CMD_TOGGLE   = "TOGGLE" // toggle play or pause
	CMD_NEXT     = "NEXT"
	CMD_PREVIOUS = "PREVIOUS"
)

// TODO: add reading command via unix socket for debugging

type Player struct {
	Command         chan string
	audioSourceList *list.List
	cancelfunc      context.CancelCauseFunc
	current         *list.Element
	toggleCh        chan bool
}

type WavMetadata struct {
	bytesPerSample int
	sampleRate     int
	channelNum     int
}

var cancelReasonNext = errors.New("next")
var cancelReasonPause = errors.New("pause")

func NewPlayer() (*Player, error) {
	audioSourceList := list.New()
	err := CreateAudioSourceList(audioSourceList, DATADIR)
	if err != nil {
		return nil, err
	}
	slog.Debug("gathered files", "len", audioSourceList.Len())
	return &Player{
		audioSourceList: audioSourceList,
		Command:         make(chan string),
		current:         audioSourceList.Front(),
		toggleCh:        make(chan bool),
	}, nil
}

func (player *Player) isPaused() bool {
	return player.cancelfunc == nil
}

func (player *Player) getCurrentAudioSource() *AudioSource {
	if player.current != nil {
		as, ok := player.current.Value.(*AudioSource)
		if ok {
			return as
		}
	}
	return nil
}

func (player *Player) setCurrentsNext() {
	if player.current != nil {
		player.current = player.current.Next()
	}
	if player.current == nil {
		player.current = player.audioSourceList.Front()
	}
}

func doPlay(ctx context.Context, as *AudioSource) error {
	slog.Debug("doPlay begin", "AudioSource", as.String())

	wavMetadata, err := readWavMetadata(as.path)
	if err != nil {
		return err
	}
	// XXX: alsaplayer discards mono wav files, but it seems to work anyways
	// FIXME: correct fix would be to convert it to stereo on-the-fly
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
		// TODO: button pushes don't work for some NewPlayer errors? ... further investigating needed
		return err
	}
	defer alsaplayer.Close()

	file, err := os.Open(as.path)
	if err != nil {
		return err
	}
	defer file.Close()

	if as.offset != 0 {
		_, err := file.Seek(as.offset, 0)
		if err != nil {
			return err
		}
		slog.Debug("continue paused title", "AudioSource", as.String())
	}

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	if err == context.Canceled && context.Cause(ctx) == cancelReasonPause {
		as.offset = as.offset + written_bytes
	} else {
		as.offset = 0
	}
	return err
}

func (player *Player) executeCancel(cause error) bool {
	if player.cancelfunc != nil {
		player.cancelfunc(cause)
		player.cancelfunc = nil
		return true
	}
	return false
}

func readWavMetadata(path string) (*WavMetadata, error) {
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

func (player *Player) Play() {
	for {
		<-player.toggleCh

		for {
			as := player.getCurrentAudioSource()
			if as == nil {
				slog.Error("could not fetch current AudioSource")
				os.Exit(1)
			}
			ctx, cancelfunc := context.WithCancelCause(context.Background())
			player.cancelfunc = cancelfunc
			err := doPlay(ctx, as)
			if err == context.Canceled {
				slog.Debug("interrupt/cancelation", "AudioSource", as.String())
				break
			} else if err != nil {
				slog.Error("doPlay() failed", "AudioSource", as.String(), "error", err)
			}
			player.setCurrentsNext()
		}
	}
}

func (player *Player) Toggle() {
	if !player.isPaused() {
		player.executeCancel(cancelReasonPause)
	} else {
		player.toggleCh <- true
	}
}

func (player *Player) Next() {
	player.executeCancel(cancelReasonNext)

	as := player.getCurrentAudioSource()
	if as == nil {
		slog.Error("current title can not be detected")
		return
	}
	player.setCurrentsNext()
	player.toggleCh <- true
}

func (player *Player) Previous() {
	slog.Error("TODO: implement Player.Previous")
}

func (player *Player) Run() {
	go player.Play()
	for {
		slog.Debug("wait for command")
		command := <-player.Command
		slog.Debug("received command", "command", command)
		switch command {
		case CMD_NEXT:
			player.Next()
		case CMD_PREVIOUS:
			player.Previous()
		case CMD_TOGGLE:
			player.Toggle()
		default:
			slog.Error("unknown command", "command", command)
		}
	}
}
