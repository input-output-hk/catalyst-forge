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

// parallelCopy performs parallel copying from multiple sources
type parallelCopy struct {
	writers []io.Writer
	errors  []error
	wg      sync.WaitGroup
	mu      sync.Mutex
}

// newParallelCopy creates a new parallel copy coordinator
func newParallelCopy(writers ...io.Writer) *parallelCopy {
	return &parallelCopy{
		writers: writers,
		errors:  make([]error, len(writers)),
	}
}

// Write implements io.Writer, distributing writes to all writers in parallel
func (pc *parallelCopy) Write(p []byte) (n int, err error) {
	pc.wg.Add(len(pc.writers))
	
	// Copy data to all writers in parallel
	data := make([]byte, len(p))
	copy(data, p)
	
	for i, w := range pc.writers {
		go func(idx int, writer io.Writer) {
			defer pc.wg.Done()
			
			_, writeErr := writer.Write(data)
			if writeErr != nil {
				pc.mu.Lock()
				pc.errors[idx] = writeErr
				pc.mu.Unlock()
			}
		}(i, w)
	}
	
	pc.wg.Wait()
	
	// Check for errors
	for _, e := range pc.errors {
		if e != nil {
			return 0, e
		}
	}
	
	return len(p), nil
}

// streamWithProgress wraps a reader to report progress
type streamWithProgress struct {
	r        io.Reader
	total    int64
	current  int64
	callback func(current, total int64)
	mu       sync.Mutex
}

// newStreamWithProgress creates a progress-reporting reader
func newStreamWithProgress(r io.Reader, total int64, callback func(current, total int64)) io.Reader {
	return &streamWithProgress{
		r:        r,
		total:    total,
		callback: callback,
	}
}

func (sp *streamWithProgress) Read(p []byte) (n int, err error) {
	n, err = sp.r.Read(p)
	
	if n > 0 && sp.callback != nil {
		sp.mu.Lock()
		sp.current += int64(n)
		current := sp.current
		sp.mu.Unlock()
		
		sp.callback(current, sp.total)
	}
	
	return n, err
}

// pooledBuffer manages a pool of reusable buffers
type pooledBuffer struct {
	pool *sync.Pool
}

// newPooledBuffer creates a new buffer pool
func newPooledBuffer(size int) *pooledBuffer {
	return &pooledBuffer{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (pb *pooledBuffer) Get() []byte {
	return pb.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (pb *pooledBuffer) Put(buf []byte) {
	// Clear sensitive data before returning to pool
	for i := range buf {
		buf[i] = 0
	}
	pb.pool.Put(buf)
}

// teeReader duplicates reads to multiple destinations
type teeReader struct {
	r       io.Reader
	writers []io.Writer
}

// newTeeReader creates a reader that duplicates to multiple writers
func newTeeReader(r io.Reader, writers ...io.Writer) io.Reader {
	return &teeReader{
		r:       r,
		writers: writers,
	}
}

func (tr *teeReader) Read(p []byte) (n int, err error) {
	n, err = tr.r.Read(p)
	if n > 0 {
		for _, w := range tr.writers {
			if _, werr := w.Write(p[:n]); werr != nil && err == nil {
				err = werr
			}
		}
	}
	return n, err
}