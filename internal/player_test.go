package godible

import (
	"bytes"
	"container/list"
	"crypto/sha256"
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

func doTestFileList(t *testing.T, audioMediumList *list.List, root string) {
	err := GatherAudioMediumsDir(audioMediumList, root)
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
		am, ok := element.Value.(*AudioMedium)
		if !ok {
			t.Fatalf("expected *AudioMedium; is %+v", element.Value)
		}
		amPath := am.GetPath()
		if amPath == path {
			return true
		}
		element = element.Next()
	}
}

func TestFileList(t *testing.T) {
	tmpBaseDir := t.TempDir()
	err := os.MkdirAll(tmpBaseDir+"/d/dd/ddd/dddd/ddddd", 0750)
	if err != nil {
		t.Fatalf("creating test directories failed")
	}
	regFiles := []string{
		"/f0",
		"/f1",
		"/d/f2",
		"/d/f3",
		"/d/dd/f4",
		"/d/dd/f5",
		"/d/dd/ddd/f6",
		"/d/dd/ddd/f7",
		"/d/dd/ddd/dddd/ddddd/f8",
	}
	for _, subPath := range regFiles {
		file, err := os.OpenFile(tmpBaseDir+subPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer closeFile(t, file)
		if _, err := file.Write([]byte(subPath)); err != nil {
			t.Fatal(err)
		}
	}

	fileList := list.New()
	doTestFileList(t, fileList, tmpBaseDir)
	if fileList.Len() != len(regFiles) {
		t.Errorf("expected list with %d entries; got list with %d entries", fileList.Len(), len(regFiles))
	}
	for _, file := range regFiles {
		if !listContainsPath(t, fileList, tmpBaseDir+file) {
			t.Errorf("list did not contain: %s", tmpBaseDir+file)
		}
	}
}

func TestFileHashes(t *testing.T) {
	tmpBaseDir := t.TempDir()
	content := []byte("hello\n")
	err := os.WriteFile(tmpBaseDir+"/file", content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	hash.Write(content)
	expectedChecksum := hash.Sum(nil)

	fileList := list.New()
	doTestFileList(t, fileList, tmpBaseDir)
	am, _ := fileList.Front().Value.(*AudioMedium)
	isChecksum := am.GetChecksum()
	if !bytes.Equal(expectedChecksum, isChecksum) {
		t.Errorf("expected checksum to be %x, is %x", expectedChecksum, isChecksum)
	}
}

func TestExoticPaths(t *testing.T) {
	tmpBaseDir := t.TempDir()

	path_str := "/a[ ]b/c /d/"
	err := os.MkdirAll(tmpBaseDir+path_str, 0750)
	if err != nil {
		t.Fatalf("creating test directories failed")
	}

	content := []byte("hello\n")
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
	am, _ := fileList.Front().Value.(*AudioMedium)
	isPath := am.GetPath()
	if filepath.Clean(isPath) != filepath.Clean(expectedPath) {
		t.Errorf("expected Path to be %s, is %s", expectedPath, isPath)
	}
}
