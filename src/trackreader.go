package godible

import (
	"fmt"
	"log/slog"
	"os"
)

type TrackReader interface {
	Read(p []byte) (n int, err error)
	Close() error
}

type WavReader struct {
	file *os.File
}

func (w WavReader) Read(p []byte) (n int, err error) {
	return w.file.Read(p)
}

func (w WavReader) Close() error {
	return w.file.Close()
}

func wavTrackReader(track *Track) (TrackReader, error) {
	file, err := os.Open(track.path)
	if err != nil {
		return nil, err
	}

	if track.offset != 0 {
		_, err := file.Seek(track.offset, 0)
		if err != nil {
			return nil, err
		}
		slog.Debug("continue paused title", "Track", track.String())
	}

	wavReader := WavReader{file: file}
	return wavReader, nil

}

func oggTrackReader(track *Track) (TrackReader, error) {
	_ = track
	return nil, fmt.Errorf("TODO: implement me")
}

func mp3TrackReader(track *Track) (TrackReader, error) {
	_ = track
	return nil, fmt.Errorf("TODO: implement me")
}

func NewTrackReader(track *Track) (TrackReader, error) {
	switch track.metadata.audioFormat {
	case MP3:
		return mp3TrackReader(track)
	case OGG:
		return oggTrackReader(track)
	default:
		return wavTrackReader(track)
	}
}
