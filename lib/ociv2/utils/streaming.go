package utils

import (
	"io"
	"sync"
)

// BufferedReader wraps an io.Reader with a buffer for efficient reading
type BufferedReader struct {
	R   io.Reader
	Buf []byte
	mu  sync.Mutex
}

func (br *BufferedReader) Read(p []byte) (n int, err error) {
	br.mu.Lock()
	defer br.mu.Unlock()

	// If request is larger than buffer, read directly
	if len(p) >= len(br.Buf) {
		return br.R.Read(p)
	}

	// Use buffer for small reads
	n, err = br.R.Read(br.Buf)
	if n > 0 {
		copy(p, br.Buf[:n])
		if n > len(p) {
			n = len(p)
		}
	}
	return n, err
}
