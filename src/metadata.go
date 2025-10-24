package godible

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-audio/wav"
	"github.com/h2non/filetype"
)

type AudioFileFormat int

const (
	WAV AudioFileFormat = iota
	MP3
	OGG
	UNKNOWN
)

type Metadata struct {
	audioFormat    AudioFileFormat
	bytesPerSample int
	sampleRate     int
	channelNum     int
}

func wavMetadata(f *os.File) (*Metadata, error) {
	d := wav.NewDecoder(f)
	d.ReadInfo()
	err := d.Err()
	if err != nil {
		return nil, err
	}
	numChans := d.NumChans
	if numChans != 2 {
		// XXX: alsaplayer discards mono wav files, but it seems to work anyways
		slog.Info("number of channels is unsupported, try to enforce 2", "NumChans", d.NumChans)
		numChans = 2
	}
	return &Metadata{
		audioFormat:    WAV,
		bytesPerSample: int(d.SampleBitDepth() / 8),
		sampleRate:     int(d.SampleRate),
		channelNum:     int(numChans),
	}, nil

}

func mp3Metadata(f *os.File) (*Metadata, error) {
	_ = f
	return nil, fmt.Errorf("TODO: implement me")
}

func oggMetadata(f *os.File) (*Metadata, error) {
	_ = f
	return nil, fmt.Errorf("TODO: implement me")
}

func detectAudioFileFormat(path string) (AudioFileFormat, error) {
	f, err := os.Open(path)
	if err != nil {
		return UNKNOWN, err
	}
	// only first 261 bytes representing the max file header is required
	head := make([]byte, 261)
	_, err = f.Read(head)
	if err != nil {
		return UNKNOWN, err
	}
	kind, _ := filetype.Match(head)
	switch kind.Extension {
	case "mp3":
		return MP3, nil
	case "wav":
		return WAV, nil
	case "ogg":
		return OGG, nil
	default:
		return UNKNOWN, fmt.Errorf("unsupported file format: %s", kind.Extension)
	}
}

func NewMetadata(path string) (*Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	af, err := detectAudioFileFormat(path)
	if err != nil {
		return nil, err
	}
	switch af {
	case MP3:
		return mp3Metadata(f)
	case OGG:
		return oggMetadata(f)
	default:
		return wavMetadata(f)

	}
}
