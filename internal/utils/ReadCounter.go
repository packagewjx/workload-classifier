package utils

import "io"

type ReadCounter struct {
	Count  int
	Reader io.Reader
}

func (r ReadCounter) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Count += n
	return
}
