package godible

import (
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

type AudioMedium struct {
	Path     string
	offset   int64  // io#Seeker.Seek
	size     int64  // fs#FileInfo.Size
	Checksum []byte // hash#Hash.Sum
}

// TODO: quit chan/context/sync-object?
type Player struct {
	alsaplayer *alsa.Player
	current    *AudioMedium
	queue      chan *AudioMedium
	playing    bool
	// TODO: replace array with container/list
	list []*AudioMedium
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
func GatherAudioMediumsDir(root string) ([]*AudioMedium, error) {
	var ret []*AudioMedium

	// check if file exists or other error occurs
	root_fileinfo, err := os.Stat(root)
	if err != nil {
		return ret, err
	}
	if !root_fileinfo.Mode().IsDir() {
		return ret, fmt.Errorf("player: given path is not a directory: %s", root)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return ret, err
	}
	for _, entry := range entries {
		entry_path := root + "/" + entry.Name()
		if entry.IsDir() {
			nestedFiles, err := GatherAudioMediumsDir(entry_path)
			if err != nil {
				// TODO: handle graciously
				continue
			}
			ret = append(ret, nestedFiles...)
			continue
		}

		if entry.Type().IsRegular() {
			checksum, err := fileHash(entry_path)
			if err != nil {
				// TODO: handle graciously
				continue
			}
			audioMedium := &AudioMedium{
				Path:     entry_path,
				Checksum: checksum,
			}
			ret = append(ret, audioMedium)
		}
	}
	return ret, nil
}

func NewPlayer() (*Player, error) {
	alsaPlayer, err := alsa.NewPlayer(44100, 2, 2, 4096)
	if err != nil {
		return nil, err
	}
	list, err := GatherAudioMediumsDir(DATADIR)
	if err != nil {
		return nil, err
	}
	return &Player{
		alsaplayer: alsaPlayer,
		list:       list,
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
	select {
	case am := <-player.queue:
		player.current = am
	default:
		if len(player.list) != 0 {
			player.current = player.list[0]
		}
	}
}

func (player *Player) getCurrent() *AudioMedium {
	if player.current != nil {
		player.setCurrent()
	}
	return player.current
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
		current := player.getCurrent()
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
