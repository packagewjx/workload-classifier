package utils

import "io"

type WriterCounter struct {
	Writer io.Writer
	Count  uint64
}

func (w *WriterCounter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	w.Count += uint64(n)
	return
}
