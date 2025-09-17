package godible

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"
)

func closeFile(t *testing.T, file *os.File) {
	err := file.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func doTestFileList(t *testing.T, baseDir string) []*AudioMedium {
	list, err := GatherAudioMediumsDir(baseDir)
	if err != nil {
		t.Errorf("dir %s; unexpected error: %s", baseDir, err)
	}
	return list
}

func listContainsPath(list []*AudioMedium, path string) bool {
	for _, audioMedium := range list {
		if audioMedium.Path == path {
			return true
		}
	}
	return false
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

	list := doTestFileList(t, tmpBaseDir)
	if len(list) != len(regFiles) {
		t.Errorf("expected list with %d entries; got list with %d entries", len(list), len(regFiles))
	}
	for _, file := range regFiles {
		if !listContainsPath(list, tmpBaseDir+file) {
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

	list := doTestFileList(t, tmpBaseDir)
	if !bytes.Equal(expectedChecksum, list[0].Checksum) {
		t.Errorf("expected checksum to be %x, is %x", expectedChecksum, list[0].Checksum)
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
	err = os.WriteFile(tmpBaseDir+path_str+"/file", content, 0644)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	hash.Write(content)
	expectedChecksum := hash.Sum(nil)

	list := doTestFileList(t, tmpBaseDir)
	if len(list) != 1 {
		t.Errorf("expected a file in gathered list")
	}
	if !bytes.Equal(expectedChecksum, list[0].Checksum) {
		t.Errorf("expected checksum to be %x, is %x", expectedChecksum, list[0].Checksum)
	}
}
