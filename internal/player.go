package godible

import (
	"container/list"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/anisse/alsa"
)

const (
	DATADIR = "/perm/godible-data/"
)

// TODO: add reading command via unix socket for debugging

type AudioMedium struct {
	Path string
	// TODO: implement pause/play logic with offset
	offset int64 // io#Seeker.Seek
	// TODO: implement size
	size     int64  // fs#FileInfo.Size
	Checksum []byte // hash#Hash.Sum
}

type Player struct {
	current         *list.Element
	queue           chan *list.Element
	playing         bool
	audioMediumList *list.List
	stop            context.CancelFunc
}

func (am *AudioMedium) String() string {
	if am == nil {
		return "nil"
	}
	return fmt.Sprintf("{Path: %s, offset: %d, size: %d, Checksum: %x}", am.Path, am.offset, am.size, am.Checksum)
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
				Path:     entry_path,
				Checksum: checksum,
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
	slog.Debug("doPlay begin", "path", am.Path, "offset", am.offset, "size", am.size)

	// FIXME: first open file, then check samplerate, and then initialize player with 44100 or 48000 Hz
	alsaplayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		slog.Error("alsaplayer could not be initialized", "err", err)
		return
	}
	defer alsaplayer.Close()

	file, err := os.Open(am.Path)
	if err != nil {
		slog.Error("file can not be opened", "path", am.Path, "err", err)
		return
	}
	defer file.Close()

	// alsaplayer.Write is not abortable/interruptable.
	// io.Copy (as in copyctx.go of github.com/anisse/alsa) failed with 'short write' (always 2 bytes short)
	// Therefore, our own WriteCtx is interruptable by introducing a
	// contexed, oldschool, buffered write.
	written_bytes, err := WriteCtx(ctx, alsaplayer, file)
	if err != nil {
		switch err {
		case context.Canceled:
			slog.Debug("stop playing of current title", "path", am.Path, "offset", written_bytes, "size", am.size)
			am.offset = written_bytes
		default:
			slog.Error("playing failed", "path", am.Path, "err", err)
		}
	}
	slog.Debug("doPlay done", "path", am.Path, "offset", am.offset, "size", am.size)
}

func (player *Player) Play(ctx context.Context) {
	am := player.getCurrentAudioMedium()
	if am == nil {
		slog.Error("Play failed: current title can not be detected")
		return
	}
	doPlay(ctx, am)
}

func (player *Player) Run() {
	// TODO create a select state machine with one waitgroup
	// - pause/play
	// - previous
	// - next
	for {
		if player.current == nil {
			slog.Info("Play: try to determine unset current title")
			player.setCurrent()
			time.Sleep(1 * time.Second)
			player.setCurrent()
			continue
		}
		current, _ := player.current.Value.(*AudioMedium)
		slog.Info("Play: determined current title", "current", current.Path)
		ctx, stop := context.WithCancel(context.Background())
		player.stop = stop
		player.Play(ctx)
		player.stop = nil
		player.current = player.current.Next()
	}
}

// FIXME: Stop() is currently more like a "Next" function
func (player *Player) Stop() {
	if player.stop != nil {
		player.stop()
	}
}

func (player *Player) Next() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
}

func (player *Player) Previous() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
}
