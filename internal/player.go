package godible

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

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

type Player struct {
	player  *alsa.Player
	current *AudioMedium
	queue   chan *AudioMedium
	playing bool
	list    []*AudioMedium
}

func fileHash(filepath string) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// TODO: implement recursive file/dir watch,, e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
func GatherAudioMediumsDir(root string) ([]*AudioMedium, error) {
	var ret []*AudioMedium

	// check if file exists or other error occurs
	basePath_fileinfo, err := os.Stat(root)
	if err != nil {
		return ret, err
	}
	if !basePath_fileinfo.Mode().IsDir() {
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
		player: alsaPlayer,
		list:   list,
	}, nil
}

func (player *Player) Next() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
}

func (player *Player) Previous() (*AudioMedium, error) {
	// TODO: implement me
	return nil, nil
}

func (player *Player) Play() {
	// TODO: implement me
}

func (player *Player) Pause() {
	// TODO: implement me
}

func (player *Player) TogglePlayPause() {
	// TODO: implement me
}
