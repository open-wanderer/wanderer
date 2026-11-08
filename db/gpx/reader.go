package gpx

import (
	"bufio"
	"io"
)

// readBufferSize is the size of the buffer between the cap reader and the
// decoder.
const readBufferSize = 64 << 10

// maxEmptyReads bounds how often the probe tolerates a (0, nil) read.
const maxEmptyReads = 100

// capReader refuses to deliver more than limit bytes. Once limit bytes have
// been read it asks the source for one probe byte: data means the input is
// over the limit and the read fails with ErrTooLarge, anything else is the
// source's own error or EOF. A document of exactly limit bytes therefore
// passes, and the source is never read more than one byte past the limit.
type capReader struct {
	r        io.Reader
	limit, n int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.n >= c.limit {
		var probe [1]byte
		for i := 0; i < maxEmptyReads; i++ {
			k, err := c.r.Read(probe[:])
			if k > 0 {
				return 0, ErrTooLarge
			}
			if err != nil {
				return 0, err
			}
		}
		return 0, io.ErrNoProgress
	}
	if rem := c.limit - c.n; int64(len(p)) > rem {
		p = p[:rem]
	}
	k, err := c.r.Read(p)
	c.n += int64(k)
	return k, err
}

// newPipeline wraps r in the byte cap and a read buffer. A non-positive
// maxBytes is a zero-byte budget.
//
// Every byte reaches the decoder through this function. The stages that
// follow the cap, in order, are inserted between the cap reader and the
// buffer: a BOM and UTF-16 sniff, the declared charset, a UTF-8 and XML
// character sanitizer, and a guard on the longest run without a '<' (the
// decoder buffers a whole run before it returns it). The order matters:
// repairing UTF-8 before the charset decode would destroy Latin-1 text.
func newPipeline(r io.Reader, maxBytes int64) io.Reader {
	if maxBytes < 0 {
		maxBytes = 0
	}
	return bufio.NewReaderSize(&capReader{r: r, limit: maxBytes}, readBufferSize)
}
