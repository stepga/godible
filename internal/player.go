package godible

import (
	"container/list"
	"context"
	"log/slog"
	"os"

	"github.com/anisse/alsa"
)

const (
	DATADIR      = "/perm/godible-data/"
	CMD_TOGGLE   = "TOGGLE" // toggle play or pause
	CMD_QUIT     = "QUIT"
	CMD_NEXT     = "NEXT"
	CMD_PREVIOUS = "PREVIOUS"
)

// TODO: add reading command via unix socket for debugging

type Player struct {
	current         *list.Element
	queue           chan *list.Element
	audioSourceList *list.List
	cancelfunc      context.CancelFunc
	Command         chan string
}

// TODO: add sync.Mutex to Player, to synchronize all access/writes to audioSourceList

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
	}, nil
}

func (player *Player) setCurrent() {
	if player.current != nil {
		return
	}

	select {
	case as := <-player.queue:
		player.current = as
	default:
		player.current = player.audioSourceList.Front()
	}
}

func (player *Player) getCurrent() *AudioSource {
	player.setCurrent()
	if player.current != nil {
		as, ok := player.current.Value.(*AudioSource)
		if ok {
			return as
		}
	}
	return nil
}

func doPlay(ctx context.Context, as *AudioSource) {
	slog.Debug("doPlay begin", "path", as.path, "offset", as.offset, "size", as.size)

	// FIXME: first open file, then check samplerate, and then initialize player with 44100 or 48000 Hz
	alsaplayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		slog.Error("alsaplayer could not be initialized", "err", err)
		return
	}
	defer alsaplayer.Close()

	file, err := os.Open(as.path)
	if err != nil {
		slog.Error("file can not be opened", "path", as.path, "err", err)
		return
	}
	defer file.Close()

	if as.offset != 0 {
		seeked_offset, err := file.Seek(as.offset, 0)
		if err != nil {
			slog.Error("failed to continue paused title", "path", as.path, "offset", as.offset, "err", err)
		}
		slog.Debug("continue paused title", "path", as.path, "offset", as.offset, "seeked_offset", seeked_offset)
	}

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	if err != nil {
		switch err {
		case context.Canceled:
			slog.Debug("cancel playing of current title", "path", as.path, "offset", written_bytes, "size", as.size)
			as.offset = as.offset + written_bytes
		default:
			slog.Error("playing failed", "path", as.path, "err", err)
		}
	} else {
		as.offset = 0
	}
	slog.Debug("doPlay done", "path", as.path, "offset", as.offset, "size", as.size)
}

func (player *Player) Play() {
	defer func() {
		player.cancelfunc = nil
	}()

	for {
		ctx, cancelfunc := context.WithCancel(context.Background())
		player.cancelfunc = cancelfunc

		as := player.getCurrent()
		if as == nil {
			slog.Error("Play failed: current title can not be detected")
			return
		}
		doPlay(ctx, as)
		if as.offset != 0 {
			slog.Debug("Play has been interrupted", "current", as.path, "offset", as.offset)
			// paused via cancelfunc
			return
		}
		player.current = player.current.Next()
	}
}
func (player *Player) Toggle() {
	if player.cancelfunc != nil {
		player.cancelfunc()
	} else {
		player.Play()
	}
}

func (player *Player) Next() {
	if player.cancelfunc != nil {
		player.cancelfunc()
	}

	as := player.getCurrent()
	if as == nil {
		slog.Error("CMD_NEXT failed: current title can not be detected")
		return
	}
	as.offset = 0

	player.current = player.current.Next()
	player.Play()
}

func (player *Player) Previous() {
	slog.Error("TODO: implement Player.Previous")
}

func (player *Player) Quit() {
	slog.Error("TODO: implement Player.Quit")
}

func (player *Player) Run() {
	for {
		// TODO: lock needed (?), and button press cancels immediately the current operation
		slog.Debug("wait for command")
		command := <-player.Command
		slog.Debug("received command", "command", command)
		switch command {
		case CMD_NEXT:
			go player.Next()
		case CMD_PREVIOUS:
			go player.Previous()
		case CMD_QUIT:
			go player.Quit()
		case CMD_TOGGLE:
			go player.Toggle()
		default:
			slog.Error("unknown command", "command", command)
		}
	}
}
