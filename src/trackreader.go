package godible

import (
	"encoding/binary"
	"io"
	"math"
	"os"

	mp3 "github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

type TrackReader interface {
	Read(p []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
	Position() (int64, error)
	Length() (int64, error)
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

func (w WavReader) Position() (int64, error) {
	return w.file.Seek(0, io.SeekCurrent)
}

func (w WavReader) Length() (int64, error) {
	fi, err := w.file.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
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

func (m Mp3Reader) Read(p []byte) (n int, err error) {
	return m.decoder.Read(p)
}

func (m Mp3Reader) Seek(offset int64, whence int) (int64, error) {
	return m.decoder.Seek(offset, whence)
}

func (m Mp3Reader) Close() error {
	return m.file.Close()
}

func (m Mp3Reader) Position() (int64, error) {
	return m.decoder.Seek(0, io.SeekCurrent)
}

func (m Mp3Reader) Length() (int64, error) {
	return m.decoder.Length(), nil
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

type OggReader struct {
	file    *os.File
	decoder *oggvorbis.Reader
}

// borrowed from anisse/beatbox/ogg.go
func (o OggReader) Read(p []byte) (n int, err error) {
	fBuf := make([]float32, len(p)/2)
	n, err = o.decoder.Read(fBuf)
	for i := 0; i < n; i += 1 {
		val := int16(fBuf[i] * math.MaxInt16)
		binary.LittleEndian.PutUint16(p[i*2:], uint16(val))
	}
	return n * 2, err
}

func (o OggReader) Seek(position int64, whence int) (int64, error) {
	_ = whence

	err := o.decoder.SetPosition(position)
	if err != nil {
		return 0, err
	}
	return position, nil
}

func (o OggReader) Close() error {
	return o.file.Close()
}

func (o OggReader) Position() (int64, error) {
	return o.decoder.Position(), nil
}

func (o OggReader) Length() (int64, error) {
	return o.decoder.Length(), nil
}

func oggTrackReader(track *Track) (TrackReader, error) {
	file, err := os.Open(track.path)
	if err != nil {
		return nil, err
	}
	dec, err := oggvorbis.NewReader(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	return OggReader{
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
	if err != nil {
		return nil, err
	}
	return ret, err
}
