package godible

import (
	"context"
	"io"
)

func WriteCtx(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	byteBuffer := make([]byte, 1024)
	bytesWrittenTotal := 0

	for {
		select {
		case <-ctx.Done():
			return int64(bytesWrittenTotal), ctx.Err()
		default:
			bytesRead, err := io.ReadFull(src, byteBuffer)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return int64(bytesWrittenTotal), err
			}
			if bytesRead == 0 {
				return int64(bytesWrittenTotal), nil
			}

			bytesWritten, err := dst.Write(byteBuffer[:bytesRead])
			if err != nil {
				return int64(bytesWrittenTotal), err
			}
			bytesWrittenTotal = bytesWrittenTotal + bytesWritten
		}
	}
}
