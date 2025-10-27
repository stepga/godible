package godible

import (
	"fmt"
	"os"

	"github.com/go-audio/wav"
	"github.com/h2non/filetype"
	mp3 "github.com/hajimehoshi/go-mp3"
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
	return &Metadata{
		audioFormat:    WAV,
		bytesPerSample: int(d.SampleBitDepth() / 8),
		sampleRate:     int(d.SampleRate),
		channelNum:     2, // alsaplayer: enforce 2 channels, even for mono filesint(numChans),
	}, nil
}

func mp3Metadata(f *os.File) (*Metadata, error) {
	dec, err := mp3.NewDecoder(f)
	if err != nil {
		return nil, err
	}
	return &Metadata{
		audioFormat:    MP3,
		bytesPerSample: 2, // enforce 2, as the bitdepth is a feature of uncompressed audio
		sampleRate:     int(dec.SampleRate()),
		channelNum:     2, // alsaplayer: enforce 2 channels, even for mono filesint(numChans),
	}, nil
}

func oggMetadata(f *os.File) (*Metadata, error) {
	_ = f
	return nil, fmt.Errorf("TODO: implement oggMetadata")
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
