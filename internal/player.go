package godible

import (
	"container/list"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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
	audioMediumList *list.List
	cancelfunc      context.CancelFunc
	Command         chan string
}

type AudioMedium struct {
	path string
	// TODO: implement pause/play logic with offset
	offset int64 // io#Seeker.Seek
	// TODO: implement size
	size     int64  // fs#FileInfo.Size
	checksum []byte // hash#Hash.Sum
}

func fileHash(filepath string) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func fileSize(filepath string) (int64, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return 0, err
	}
	// get the size
	return fi.Size(), nil
}

func (am *AudioMedium) GetPath() string {
	return am.path
}

func (am *AudioMedium) GetChecksum() []byte {
	return am.checksum
}

func (am *AudioMedium) String() string {
	if am == nil {
		return "nil"
	}
	return fmt.Sprintf("{path: %s, offset: %d, size: %d, checksum: %x}", am.path, am.offset, am.size, am.checksum)
}

// TODO: implement recursive file/dir watch,, e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
func GatherAudioMediumsDir(audioMediumList *list.List, root string) error {
	if audioMediumList == nil {
		audioMediumList = list.New()
	}
	root_fileinfo, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !root_fileinfo.Mode().IsDir() {
		return fmt.Errorf("given path is not a directory: %s", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entry_path := root + "/" + entry.Name()
		if entry.IsDir() {
			err := GatherAudioMediumsDir(audioMediumList, entry_path)
			if err != nil {
				slog.Error("GatherAudioMediumsDir for subdirectory failed", "directory", entry_path)
			}
			continue
		}
		if entry.Type().IsRegular() {
			checksum, err := fileHash(entry_path)
			if err != nil {
				slog.Error("fileHash for regular file failed", "file", entry_path, "err", err)
				continue
			}
			filesize, err := fileSize(entry_path)
			if err != nil {
				slog.Error("fileSize for regular file failed", "file", entry_path, "err", err)
				continue
			}
			audioMedium := &AudioMedium{
				path:     entry_path,
				checksum: checksum,
				size:     filesize,
			}
			audioMediumList.PushBack(audioMedium)
		}
	}
	return nil
}

func NewPlayer() (*Player, error) {
	audioMediumList := list.New()
	err := GatherAudioMediumsDir(audioMediumList, DATADIR)
	if err != nil {
		return nil, err
	}
	slog.Debug("gathered files", "len", audioMediumList.Len())
	return &Player{
		audioMediumList: audioMediumList,
		Command:         make(chan string),
	}, nil
}

func (player *Player) setCurrent() {
	if player.current != nil {
		return
	}

	select {
	case am := <-player.queue:
		player.current = am
	default:
		player.current = player.audioMediumList.Front()
	}
}

func (player *Player) getCurrentAudioMedium() *AudioMedium {
	player.setCurrent()
	if player.current != nil {
		am, ok := player.current.Value.(*AudioMedium)
		if ok {
			return am
		}
	}
	return nil
}

func doPlay(ctx context.Context, am *AudioMedium) {
	slog.Debug("doPlay begin", "path", am.path, "offset", am.offset, "size", am.size)

	// FIXME: first open file, then check samplerate, and then initialize player with 44100 or 48000 Hz
	alsaplayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		slog.Error("alsaplayer could not be initialized", "err", err)
		return
	}
	defer alsaplayer.Close()

	file, err := os.Open(am.path)
	if err != nil {
		slog.Error("file can not be opened", "path", am.path, "err", err)
		return
	}
	defer file.Close()

	if am.offset != 0 {
		seeked_offset, err := file.Seek(am.offset, 0)
		if err != nil {
			slog.Error("failed to continue paused title", "path", am.path, "offset", am.offset, "err", err)
		}
		slog.Debug("continue paused title", "path", am.path, "offset", am.offset, "seeked_offset", seeked_offset)
	}

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	if err != nil {
		switch err {
		case context.Canceled:
			slog.Debug("cancel playing of current title", "path", am.path, "offset", written_bytes, "size", am.size)
			am.offset = am.offset + written_bytes
		default:
			slog.Error("playing failed", "path", am.path, "err", err)
		}
	} else {
		am.offset = 0
	}
	slog.Debug("doPlay done", "path", am.path, "offset", am.offset, "size", am.size)
}

func (player *Player) Play() {
	defer func() {
		player.cancelfunc = nil
	}()

	for {
		ctx, cancelfunc := context.WithCancel(context.Background())
		player.cancelfunc = cancelfunc

		am := player.getCurrentAudioMedium()
		if am == nil {
			slog.Error("Play failed: current title can not be detected")
			return
		}
		doPlay(ctx, am)
		if am.offset != 0 {
			slog.Debug("Play has been interrupted", "current", am.path, "offset", am.offset)
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

	am := player.getCurrentAudioMedium()
	if am == nil {
		slog.Error("CMD_NEXT failed: current title can not be detected")
		return
	}
	am.offset = 0

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
