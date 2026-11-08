package gpx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// readBufferSize is the size of the buffer between the last reader stage and
// the decoder.
const readBufferSize = 64 << 10

// maxEmptyReads bounds how often the probe and the sanitizer tolerate a
// (0, nil) read.
const maxEmptyReads = 100

// The BOM and charset stages peek at the start of the document through
// buffers of stageBufferSize.
const (
	bomSniffSize     = 4
	charsetSniffSize = 512
	stageBufferSize  = 4096
)

// capReader refuses to deliver more than limit bytes. Once limit bytes have
// been read it asks the source for one probe byte: data means the input is
// over the limit and the read fails with ErrTooLarge, anything else is the
// source's own error or EOF. A document of exactly limit bytes therefore
// passes, and the source is never read more than one byte past the limit.
//
// The probe's outcome is remembered. bufio.Reader.Peek hands an error back to
// its caller and forgets it, so a later stage would otherwise probe again and
// read a second byte past the limit.
type capReader struct {
	r        io.Reader
	limit, n int64
	err      error // outcome of the probe, once it has run
}

func (c *capReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.err != nil {
		return 0, c.err
	}
	if c.n >= c.limit {
		var probe [1]byte
		for i := 0; i < maxEmptyReads; i++ {
			k, err := c.r.Read(probe[:])
			if k > 0 {
				c.err = ErrTooLarge
				return 0, c.err
			}
			if err != nil {
				c.err = err
				return 0, c.err
			}
		}
		c.err = io.ErrNoProgress
		return 0, c.err
	}
	if rem := c.limit - c.n; int64(len(p)) > rem {
		p = p[:rem]
	}
	k, err := c.r.Read(p)
	c.n += int64(k)
	return k, err
}

// encodingDecl finds the encoding of an XML declaration at the very start of
// the (BOM-free) text.
var encodingDecl = regexp.MustCompile(`^<\?xml[^>]*?encoding\s*=\s*["']([^"']+)["']`)

// newPipeline turns r into a stream of valid UTF-8 XML text for the decoder.
// A non-positive maxBytes is a zero-byte budget.
//
// Every byte reaches the decoder through this function, in this order:
//
//  1. the byte cap;
//  2. a BOM and UTF-16 sniff;
//  3. the declared charset;
//  4. the sanitizer, which repairs invalid UTF-8 and XML-illegal characters;
//  5. the run guard, which bounds the longest run without a '<' (the decoder
//     buffers a whole run before it returns it).
//
// The order matters: repairing UTF-8 before the charset decode would destroy
// Latin-1 text. The decoder's own CharsetReader stays the identity, because by
// the time it sees the XML declaration the bytes are already UTF-8.
//
// A short Peek in stages 2 and 3 ignores its error: the cap reader remembers
// it and returns it to the next Read, so ErrTooLarge still surfaces.
func newPipeline(r io.Reader, maxBytes int64) io.Reader {
	if maxBytes < 0 {
		maxBytes = 0
	}
	// Stage 1.
	br := bufio.NewReaderSize(&capReader{r: r, limit: maxBytes}, stageBufferSize)

	// Stage 2. The sniff is by hand: unicode.BOMOverride falls back lossily.
	var in io.Reader = br
	head, _ := br.Peek(bomSniffSize)
	switch {
	case bytes.HasPrefix(head, []byte{0xEF, 0xBB, 0xBF}):
		_, _ = br.Discard(3)
	case bytes.HasPrefix(head, []byte{0xFF, 0xFE}), bytes.HasPrefix(head, []byte{0xFE, 0xFF}):
		in = transform.NewReader(br, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	}

	// Stage 3.
	br2 := bufio.NewReaderSize(in, stageBufferSize)
	decl, _ := br2.Peek(charsetSniffSize)
	in = br2
	if m := encodingDecl.FindSubmatch(decl); m != nil {
		switch label := string(bytes.ToLower(m[1])); label {
		case "utf-8", "utf8", "utf-16", "utf-16le", "utf-16be":
			// Already UTF-8, or already decoded by stage 2.
		default:
			// An unknown label, and the WHATWG "replacement" encodings that
			// decode everything to U+FFFD, keep the bytes as UTF-8 (A6).
			if _, name := charset.Lookup(label); name != "" && name != "replacement" {
				if dec, err := charset.NewReaderLabel(label, in); err == nil {
					in = dec
				}
			}
		}
	}

	// Stages 4 and 5, with a buffer so the decoder can read bytes directly.
	return bufio.NewReaderSize(&runGuard{r: &sanitizer{src: in}, max: maxRun}, readBufferSize)
}

// sanitizer produces valid XML text from arbitrary bytes: an invalid UTF-8
// sequence becomes U+FFFD, a C0 control other than tab, LF and CR becomes a
// space, and U+FFFE and U+FFFF become U+FFFD. An incomplete rune at the end of
// a Read is carried into the next one, and a partial rune left at EOF becomes
// one U+FFFD. The source's error is returned unchanged once the repaired text
// before it has been delivered.
type sanitizer struct {
	src io.Reader
	in  [stageBufferSize]byte
	n   int    // carried bytes at the start of in
	out []byte // repaired text not yet handed out
	buf []byte // backing array of out
	err error
}

func (s *sanitizer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	for empties := 0; len(s.out) == 0; {
		if s.err != nil {
			return 0, s.err
		}
		k, err := s.src.Read(s.in[s.n:])
		s.n += k
		if k == 0 && err == nil {
			if empties++; empties >= maxEmptyReads {
				s.err = io.ErrNoProgress
			}
			continue
		}
		empties = 0
		s.err = err
		s.process(err != nil)
	}
	k := copy(p, s.out)
	s.out = s.out[k:]
	return k, nil
}

// process repairs the buffered bytes into out. Unless final is set, an
// incomplete trailing rune stays in the buffer.
func (s *sanitizer) process(final bool) {
	if s.buf == nil {
		s.buf = make([]byte, 0, 3*stageBufferSize)
	}
	out := s.buf[:0]
	data := s.in[:s.n]
	i := 0
	for i < len(data) {
		c := data[i]
		if c < utf8.RuneSelf {
			if c < 0x20 && c != '\t' && c != '\n' && c != '\r' {
				c = ' '
			}
			out = append(out, c)
			i++
			continue
		}
		if !utf8.FullRune(data[i:]) {
			if !final {
				break
			}
			out = utf8.AppendRune(out, utf8.RuneError)
			i = len(data)
			break
		}
		r, size := utf8.DecodeRune(data[i:])
		switch {
		case r == utf8.RuneError && size == 1, r == 0xFFFE, r == 0xFFFF:
			out = utf8.AppendRune(out, utf8.RuneError)
		default:
			out = append(out, data[i:i+size]...)
		}
		i += size
	}
	s.n = copy(s.in[:], data[i:])
	s.out = out
}

// runGuard fails with ErrMalformed once a run of bytes without '<' is longer
// than max. The run is the bytes since the last '<' and is carried across
// Read calls. The decoder buffers names, attribute values and text whole, so
// without this guard one such run could grow without bound inside the cap.
type runGuard struct {
	r   io.Reader
	max int
	run int
}

func (g *runGuard) Read(p []byte) (int, error) {
	n, err := g.r.Read(p)
	for i, c := range p[:n] {
		if c == '<' {
			g.run = 0
			continue
		}
		if g.run++; g.run > g.max {
			return i, fmt.Errorf("%w: text run too long", ErrMalformed)
		}
	}
	return n, err
}
