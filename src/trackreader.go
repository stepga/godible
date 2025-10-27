package godible

import (
	"fmt"
	"log/slog"
	"os"

	mp3 "github.com/hajimehoshi/go-mp3"
)

type TrackReader interface {
	Read(p []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

func oggTrackReader(track *Track) (TrackReader, error) {
	_ = track
	return nil, fmt.Errorf("TODO: implement oggTrackReader")
}

type WavReader struct {
	file *os.File
}

func (w WavReader) Read(p []byte) (n int, err error) {
	return w.file.Read(p)
}

func (w WavReader) Seek(offset int64, whence int) (int64, error) {
	return w.file.Seek(offset, whence)
}

func (w WavReader) Close() error {
	return w.file.Close()
}

func wavTrackReader(track *Track) (TrackReader, error) {
	file, err := os.Open(track.path)
	if err != nil {
		return nil, err
	}
	return WavReader{file: file}, nil
}

type Mp3Reader struct {
	file    *os.File
	decoder *mp3.Decoder
}

func (w Mp3Reader) Read(p []byte) (n int, err error) {
	return w.decoder.Read(p)
}

func (w Mp3Reader) Seek(offset int64, whence int) (int64, error) {
	return w.decoder.Seek(offset, whence)
}

func (w Mp3Reader) Close() error {
	return w.file.Close()
}

func mp3TrackReader(track *Track) (TrackReader, error) {
	file, err := os.Open(track.path)
	if err != nil {
		return nil, err
	}
	dec, err := mp3.NewDecoder(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return Mp3Reader{
		file:    file,
		decoder: dec,
	}, nil
}

func NewTrackReader(track *Track) (TrackReader, error) {
	var ret TrackReader
	var err error

	switch track.metadata.audioFormat {
	case MP3:
		ret, err = mp3TrackReader(track)
	case OGG:
		ret, err = oggTrackReader(track)
	default:
		ret, err = wavTrackReader(track)
	}
	if err == nil && track.offset != 0 {
		_, err := ret.Seek(track.offset, 0)
		if err != nil {
			return nil, err
		}
		slog.Debug("continue paused title", "Track", track.String())
	}
	return ret, err
}
