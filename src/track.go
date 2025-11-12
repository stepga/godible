package godible

import (
	"container/list"
	"fmt"
	"log/slog"
	"os"
)

type Track struct {
	path     string
	position int64
	length   int64
	metadata *Metadata
	paused   bool
}

func (t *Track) GetPath() string {
	if t == nil {
		return ""
	}
	return t.path
}

func (t *Track) GetPosition() int64 {
	if t == nil {
		return -1
	}
	return t.position
}

func (t *Track) GetLength() int64 {
	if t == nil {
		return -1
	}
	return t.length
}

func (t *Track) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("{path: %s, position: %d, length: %d}", t.path, t.position, t.length)
}

func isRegularFile(path string) (bool, error) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileinfo.Mode().IsRegular(), nil
}

func NewTrack(path string) (*Track, error) {
	ok, err := isRegularFile(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}

	metadata, err := NewMetadata(path)
	if err != nil {
		return nil, err
	}
	t := Track{
		path:     path,
		metadata: metadata,
	}

	reader, err := NewTrackReader(&t)
	if err == nil {
		t.length, err = reader.Length()
	}
	if err != nil {
		slog.Error("failed to gather track's length", "err", err)
	}
	if reader != nil {
		reader.Close()
	}

	return &t, nil
}

// Creates a list of Tracks for all regular files within the given root
// directory and its subdirectories of any level.
//
// The function returns any occuring error immediately.
func CreateTrackList(tl *list.List, root string) error {
	if tl == nil {
		tl = list.New()
	}
	fileinfo, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !fileinfo.Mode().IsDir() {
		return fmt.Errorf("given path is not a directory: %s", root)
	}
	direntries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, direntry := range direntries {
		path := root + "/" + direntry.Name()
		if direntry.IsDir() {
			err := CreateTrackList(tl, path)
			if err != nil {
				return err
			}
			continue
		}
		if direntry.Type().IsRegular() {
			t, err := NewTrack(path)
			if err != nil {
				slog.Error("skip track", "path", path, "error", err)
				continue
			}
			if !sampleRateSupported(t.metadata.sampleRate) {
				slog.Error("skip track: unsupported sample rate", "path", t.path, "sample rate", t.metadata.sampleRate)
				continue
			}
			tl.PushBack(t)
		}
	}
	return nil
}
