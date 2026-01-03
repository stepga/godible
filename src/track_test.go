package godible

import (
	"container/list"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func closeFile(t *testing.T, file *os.File) {
	err := file.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func doTestFileList(t *testing.T, tracklist *list.List, root string) {
	err := CreateTrackList(tracklist, root)
	if err != nil {
		t.Errorf("dir %s; unexpected error: %s", root, err)
	}
}

func listContainsPath(t *testing.T, fileList *list.List, path string) bool {
	element := fileList.Front()
	for {
		if element == nil {
			return false
		}
		track, ok := element.Value.(*Track)
		if !ok {
			t.Fatalf("expected *AudioSource; is %+v", element.Value)
		}
		amPath := track.path
		if amPath == path {
			return true
		}
		element = element.Next()
	}
}

func minimalWavFile(t *testing.T) []byte {
	// xxd -p -c0 wav.wav (wav.wav from https://github.com/mathiasbynens/small.git)
	str := "524946462400000057415645666d7420100000000100010044ac000088580100020010006461746100000000"
	enc, err := hex.DecodeString(str)
	if err != nil {
		t.Fatalf("encoding test file content failed")
	}
	return enc
}

func TestFileList(t *testing.T) {
	tmpBaseDir := t.TempDir()
	err := os.MkdirAll(tmpBaseDir+"/d/dd/ddd/dddd/ddddd", 0750)
	if err != nil {
		t.Fatalf("creating test directories failed")
	}
	regFiles := []string{
		"/f0.wav",
		"/f1.wav",
		"/d/f2.wav",
		"/d/f3.wav",
		"/d/dd/f4.wav",
		"/d/dd/f5.wav",
		"/d/dd/ddd/f6.wav",
		"/d/dd/ddd/f7.wav",
		"/d/dd/ddd/dddd/ddddd/f8.wav",
	}
	for _, subPath := range regFiles {
		file, err := os.OpenFile(tmpBaseDir+subPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer closeFile(t, file)
		if _, err := file.Write(minimalWavFile(t)); err != nil {
			t.Fatal(err)
		}
	}

	fileList := list.New()
	doTestFileList(t, fileList, tmpBaseDir)
	if fileList.Len() != len(regFiles) {
		t.Errorf("expected list with %d entries; got list with %d entries", len(regFiles), fileList.Len())
	}
	for _, file := range regFiles {
		if !listContainsPath(t, fileList, tmpBaseDir+file) {
			t.Errorf("list did not contain: %s", tmpBaseDir+file)
		}
	}
}

func TestExoticPaths(t *testing.T) {
	tmpBaseDir := t.TempDir()

	path_str := "/a[ ]b/c /d/"
	err := os.MkdirAll(tmpBaseDir+path_str, 0750)
	if err != nil {
		t.Fatalf("creating test directories failed")
	}

	content := minimalWavFile(t)
	expectedPath := tmpBaseDir + path_str + "/file"
	err = os.WriteFile(expectedPath, content, 0644)
	if err != nil {
		t.Fatal(err)
	}

	fileList := list.New()
	doTestFileList(t, fileList, tmpBaseDir)
	if fileList.Len() != 1 {
		t.Errorf("expected a file in gathered list")
	}
	track, _ := fileList.Front().Value.(*Track)
	isPath := track.path
	if filepath.Clean(isPath) != filepath.Clean(expectedPath) {
		t.Errorf("expected Path to be %s, is %s", expectedPath, isPath)
	}
}
