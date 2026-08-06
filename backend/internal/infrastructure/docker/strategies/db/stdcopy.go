package db

import (
	"fmt"
	"io"
	"time"
)

// stdCopy demuxes Docker multiplexed stdout/stderr streams
func stdCopy(dstOut, dstErr io.Writer, src io.Reader) (int64, error) {
	var written int64
	header := make([]byte, 8)
	for {
		_, err := io.ReadFull(src, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return written, err
		}

		streamType := header[0]
		count := int64(header[4])<<24 | int64(header[5])<<16 | int64(header[6])<<8 | int64(header[7])

		var w io.Writer
		switch streamType {
		case 1: // stdout
			w = dstOut
		case 2: // stderr
			w = dstErr
		default:
			w = io.Discard
		}

		n, err := io.CopyN(w, src, count)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

type streamCloser interface {
	Close()
}

// copyWithTimeout runs stdCopy in a goroutine and enforces a max timeout to prevent blocking stream reads
func copyWithTimeout(dstOut, dstErr io.Writer, src io.Reader, c streamCloser, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		_, err := stdCopy(dstOut, dstErr, src)
		done <- err
	}()

	select {
	case err := <-done:
		if c != nil {
			c.Close()
		}
		return err
	case <-time.After(timeout):
		if c != nil {
			c.Close()
		}
		return fmt.Errorf("read stream timed out after %v", timeout)
	}
}
