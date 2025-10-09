package godible

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

type AudioSource struct {
	path     string
	offset   int64  // io#Seeker.Seek
	size     int64  // fs#fileinfo.Size
	checksum []byte // hash#Hash.Sum
}

func (as *AudioSource) GetPath() string {
	return as.path
}

func (as *AudioSource) GetChecksum() []byte {
	return as.checksum
}

func (as *AudioSource) String() string {
	if as == nil {
		return "nil"
	}
	return fmt.Sprintf("{path: %s, offset: %d, size: %d, checksum: %x}", as.path, as.offset, as.size, as.checksum)
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

func NewAudioSource(path string) (*AudioSource, error) {
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
	as := AudioSource{
		path:     path,
		checksum: checksum,
		size:     size,
	}
	return &as, nil
}

// CreateAudioSourceList creates a list of AudioSources for all regular files
// within the given root directory and its subdirectories of any level.
//
// The function returns any occuring error immediately.
func CreateAudioSourceList(audioSourceList *list.List, root string) error {
	if audioSourceList == nil {
		audioSourceList = list.New()
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
			err := CreateAudioSourceList(audioSourceList, path)
			if err != nil {
				return err
			}
			continue
		}
		if direntry.Type().IsRegular() {
			as, err := NewAudioSource(path)
			if err != nil {
				return err
			}
			audioSourceList.PushBack(as)
		}
	}
	return nil
}

// TODO: implement recursive file/dir watch, e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
