package godible

import (
	"context"
	"io"
)

// TODO: change src to io.ReadSeeker Interface?
func WriteCtx(ctx context.Context, dst io.Writer, src TrackReader, track *Track) error {
	byteBuffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			bytesRead, err := io.ReadFull(src, byteBuffer)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return err
			}
			if bytesRead == 0 {
				return nil
			}

			_, err = dst.Write(byteBuffer[:bytesRead])
			if err != nil {
				return err
			}

			track.position, err = src.Position()
			if err != nil {
				return err
			}
		}
	}
}
