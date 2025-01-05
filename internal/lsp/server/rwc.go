package server

import (
	"io"
	"os"
)

type RWC struct {
	r io.ReadCloser
	w io.WriteCloser
}

// NewStdRWC creates a new RWC using standard input/output
func NewStdRWC() *RWC {
	return &RWC{
		r: os.Stdin,
		w: os.Stdout,
	}
}

// NewRWC creates a new RWC with custom reader and writer
func NewRWC(r io.ReadCloser, w io.WriteCloser) *RWC {
	return &RWC{
		r: r,
		w: w,
	}
}

func (rw *RWC) Read(p []byte) (int, error)  { return rw.r.Read(p) }
func (rw *RWC) Write(p []byte) (int, error) { return rw.w.Write(p) }
func (rw *RWC) Close() error {
	if rw.r != nil {
		if err := rw.r.Close(); err != nil {
			return err
		}
	}
	if rw.w != nil {
		return rw.w.Close()
	}
	return nil
}
