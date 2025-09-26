package godible

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"io"
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

// TODO: quit chan/context/sync-object?
type Player struct {
	alsaplayer      *alsa.Player
	current         *list.Element
	queue           chan *list.Element
	playing         bool
	audioMediumList *list.List
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
		return fmt.Errorf("player: given path is not a directory: %s", root)
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
				panic(err)
				// TODO: handle graciously
			}
			continue
		}
		if entry.Type().IsRegular() {
			checksum, err := fileHash(entry_path)
			if err != nil {
				panic(err)
				// TODO: handle graciously
			}
			audioMedium := &AudioMedium{
				Path:     entry_path,
				Checksum: checksum,
			}
			audioMediumList.PushBack(audioMedium)
		}
	}
	return nil
}

func NewPlayer() (*Player, error) {
	// TODO: both 44100 Hz and 48000 Hz are supported. re-init new player if AudioMedium requires it ...
	alsaPlayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		return nil, err
	}
	audioMediumList := list.New()
	err = GatherAudioMediumsDir(audioMediumList, DATADIR)
	if err != nil {
		return nil, err
	}
	// TODO: log audioMediumList.Len()
	return &Player{
		alsaplayer:      alsaPlayer,
		audioMediumList: audioMediumList,
	}, nil
}

func (player *Player) Close() error {
	return player.alsaplayer.Close()
}

func (player *Player) Next() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
}

func (player *Player) Previous() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
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

func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func (player *Player) Play() {
	for {
		current := player.getCurrentAudioMedium()
		if current == nil {
			// TODO: log that no current AudioMedium could be determined
			time.Sleep(1 * time.Second)
			continue
		}
		// TODO: handle offset (modify readFile to readAudioMedium)
		data, err := readFile(current.Path)
		if err != nil {
			panic(err.Error())
		}
		_, err = player.alsaplayer.Write(data)
		if err != nil {
			panic(err.Error())
		}
		player.current = player.current.Next()
	}
}

func (player *Player) Pause() {
	// TODO: implement me:
	// - set player.current.offset
	// - toggle player.playing
}

func (player *Player) TogglePlayPause() {
	// TODO: implement me
}
