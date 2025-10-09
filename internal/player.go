package godible

import (
	"container/list"
	"context"
	"log/slog"
	"os"

	"github.com/anisse/alsa"
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
	cancelfunc      context.CancelFunc
	current         *list.Element
	toggleCh        chan bool
}

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
	slog.Debug("doPlay begin", "path", as.path, "offset", as.offset, "size", as.size)

	// FIXME: first open file, then check samplerate, and then initialize player with 44100 or 48000 Hz
	alsaplayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		return err
	}
	defer alsaplayer.Close()

	file, err := os.Open(as.path)
	if err != nil {
		return err
	}
	defer file.Close()

	if as.offset != 0 {
		seeked_offset, err := file.Seek(as.offset, 0)
		if err != nil {
			return err
		}
		slog.Debug("continue paused title", "path", as.path, "offset", as.offset, "seeked_offset", seeked_offset)
	}

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	switch err {
	case context.Canceled:
		as.offset = as.offset + written_bytes // FIXME: use WithCancelCause and differ whether Next or whether paused
	case nil:
		as.offset = 0
	}
	return err
}

func (player *Player) executeCancel() bool {
	if player.cancelfunc != nil {
		player.cancelfunc()
		player.cancelfunc = nil
		return true
	}
	return false
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
			ctx, cancelfunc := context.WithCancel(context.Background())
			player.cancelfunc = cancelfunc
			err := doPlay(ctx, as)
			if err == context.Canceled {
				slog.Debug("interrupt/cancelation", "current", as.path, "offset", as.offset)
				break
			}
			player.setCurrentsNext()
		}
	}
}

func (player *Player) Toggle() {
	if !player.isPaused() {
		player.executeCancel()
	} else {
		player.toggleCh <- true
	}
}

func (player *Player) Next() {
	player.executeCancel()

	as := player.getCurrentAudioSource()
	if as == nil {
		slog.Error("current title can not be detected")
		return
	}
	// FIXME: this does not seem to work ... Next-ing songs still results in continuing them
	// FIXME: use WithCancelCause and differ whether Next or whether paused
	as.offset = 0

	player.setCurrentsNext()
	player.toggleCh <- true
}

func (player *Player) Previous() {
	slog.Error("TODO: implement Player.Previous")
}

func (player *Player) Run() {
	go player.Play()
	for {
		// TODO: lock needed (?), and button press cancels immediately the current operation
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
