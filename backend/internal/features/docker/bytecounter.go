package docker

import "io"

// byteCounter lets a streamed backup report its size without buffering the
// whole tar in memory.
type byteCounter struct {
	source    io.Reader
	readBytes int64
}

func (c *byteCounter) Read(buffer []byte) (int, error) {
	readCount, err := c.source.Read(buffer)
	c.readBytes += int64(readCount)

	return readCount, err
}
