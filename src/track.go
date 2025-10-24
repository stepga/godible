package godible

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

type Track struct {
	path     string
	offset   int64  // io#Seeker.Seek
	size     int64  // fs#fileinfo.Size
	checksum []byte // hash#Hash.Sum
}

func (t *Track) GetPath() string {
	return t.path
}

func (t *Track) GetChecksum() []byte {
	return t.checksum
}

func (t *Track) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("{path: %s, offset: %d, size: %d, checksum: %x}", t.path, t.offset, t.size, t.checksum)
}

func isRegularFile(path string) (bool, error) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileinfo.Mode().IsRegular(), nil
}

func fileChecksum(path string) ([]byte, error) {
	file, err := os.Open(path)
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

func fileSize(path string) (int64, error) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fileinfo.Size(), nil
}

func NewTrack(path string) (*Track, error) {
	ok, err := isRegularFile(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}

	checksum, err := fileChecksum(path)
	if err != nil {
		return nil, err
	}
	size, err := fileSize(path)
	if err != nil {
		return nil, err
	}
	t := Track{
		path:     path,
		checksum: checksum,
		size:     size,
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
				return err
			}
			tl.PushBack(t)
		}
	}
	return nil
}
