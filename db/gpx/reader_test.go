package gpx

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"golang.org/x/text/encoding/unicode"
)

// countingReader records how many bytes the source handed out, so the cap tests
// can prove the reader never reads further than one probe byte past the limit.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	k, err := c.r.Read(p)
	c.n += int64(k)
	return k, err
}

const rawBody = `<gpx version="1.1"><trk><name>`
const rawTail = `</name><trkseg><trkpt lat="47" lon="11"/><trkpt lat="47.001" lon="11"/></trkseg></trk></gpx>`

// rawDoc builds a GPX around raw name bytes. header is written first and may be
// empty or hold an XML declaration, a BOM, or both. name is raw, so tests can
// put single-byte legacy encodings and broken UTF-8 into it.
func rawDoc(header string, name []byte) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString(rawBody)
	b.Write(name)
	b.WriteString(rawTail)
	return b.Bytes()
}

func declare(enc string) string {
	return `<?xml version="1.0" encoding="` + enc + `"?>` + "\n"
}

func parseBytes(t *testing.T, b []byte) *Result {
	t.Helper()
	res, err := Parse(bytes.NewReader(b), 25<<20, "")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return res
}

func TestCapReader(t *testing.T) {
	doc := rawDoc("", []byte("cap"))
	size := int64(len(doc))

	t.Run("exactly maxBytes parses", func(t *testing.T) {
		src := &countingReader{r: bytes.NewReader(doc)}
		if _, err := Parse(src, size, ""); err != nil {
			t.Fatalf("a document of exactly maxBytes must parse: %v", err)
		}
		if src.n > size+1 {
			t.Errorf("read %d bytes, want at most %d", src.n, size+1)
		}
	})
	t.Run("one byte more is too large", func(t *testing.T) {
		src := &countingReader{r: bytes.NewReader(doc)}
		res, err := Parse(src, size-1, "")
		if !errors.Is(err, ErrTooLarge) || res != nil {
			t.Fatalf("got (non-nil result: %v, %v), want (nil, ErrTooLarge)", res != nil, err)
		}
		if src.n > size {
			t.Errorf("read %d bytes, want at most limit+1 = %d", src.n, size)
		}
	})
	t.Run("exactly maxBytes parses from a one-byte reader", func(t *testing.T) {
		src := iotest.OneByteReader(bytes.NewReader(doc))
		if _, err := Parse(src, size, ""); err != nil {
			t.Fatalf("got %v", err)
		}
	})
	for _, limit := range []int64{0, -5} {
		src := &countingReader{r: bytes.NewReader(doc)}
		res, err := Parse(src, limit, "")
		if !errors.Is(err, ErrTooLarge) || res != nil {
			t.Errorf("maxBytes %d: got (non-nil result: %v, %v), want (nil, ErrTooLarge)", limit, res != nil, err)
		}
		if src.n > 1 {
			t.Errorf("maxBytes %d: read %d bytes, want at most the one probe byte", limit, src.n)
		}
	}
	t.Run("limit below the charset sniff window", func(t *testing.T) {
		big := rawDoc("", bytes.Repeat([]byte("a"), 10<<10))
		src := &countingReader{r: bytes.NewReader(big)}
		res, err := Parse(src, 100, "")
		if !errors.Is(err, ErrTooLarge) || res != nil {
			t.Fatalf("got (non-nil result: %v, %v), want (nil, ErrTooLarge)", res != nil, err)
		}
		if src.n > 101 {
			t.Errorf("read %d bytes, want at most 101", src.n)
		}
	})
}

func utf16Doc(t *testing.T, order unicode.Endianness, label string) []byte {
	t.Helper()
	text := declare(label) + string(rawDoc("", []byte("h\u00e9")))
	out, err := unicode.UTF16(order, unicode.UseBOM).NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPipeline(t *testing.T) {
	cases := []struct {
		name string
		doc  []byte
		want string
	}{
		{"utf-8 bom", rawDoc("\xef\xbb\xbf"+declare("UTF-8"), []byte("caf\xc3\xa9")), "caf\u00e9"},
		{"utf-8 bom without declaration", rawDoc("\xef\xbb\xbf", []byte("caf\xc3\xa9")), "caf\u00e9"},
		{"utf-16le with bom", utf16Doc(t, unicode.LittleEndian, "utf-16"), "h\u00e9"},
		{"utf-16be with bom", utf16Doc(t, unicode.BigEndian, "UTF-16"), "h\u00e9"},
		{"declared iso-8859-1", rawDoc(declare("ISO-8859-1"), []byte("caf\xe9")), "caf\u00e9"},
		{"declared windows-1252", rawDoc(declare("windows-1252"), []byte("\x93q\x94")), "\u201cq\u201d"},
		{"declared windows-1252 in single quotes", rawDoc(`<?xml version='1.0' encoding='Windows-1252'?>`, []byte("caf\xe9")), "caf\u00e9"},
		{"unknown declared charset reads as utf-8", rawDoc(declare("x-bogus"), []byte("caf\xc3\xa9")), "caf\u00e9"},
		{"unknown declared charset repairs bad bytes", rawDoc(declare("x-bogus"), []byte("a\xffb")), "a\ufffdb"},
		{"replacement charset reads as utf-8", rawDoc(declare("iso-2022-kr"), []byte("caf\xc3\xa9")), "caf\u00e9"},
		{"utf-8 declared", rawDoc(declare("utf-8"), []byte("caf\xc3\xa9")), "caf\u00e9"},
		{"nul becomes a space", rawDoc("", []byte("a\x00b")), "a b"},
		{"0x01 becomes a space", rawDoc("", []byte("a\x01b")), "a b"},
		{"tab stays inside the name", rawDoc("", []byte("a\tb")), "a\tb"},
		{"invalid utf-8 becomes U+FFFD", rawDoc("", []byte("a\xffb")), "a\ufffdb"},
		{"noncharacter U+FFFE becomes U+FFFD", rawDoc("", []byte("a\xef\xbf\xbeb")), "a\ufffdb"},
		{"noncharacter U+FFFF becomes U+FFFD", rawDoc("", []byte("a\xef\xbf\xbfb")), "a\ufffdb"},
		{"latin-1 text is decoded before utf-8 repair", rawDoc(declare("iso-8859-1"), []byte("\xe9\xe8\xe0")), "\u00e9\u00e8\u00e0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := parseBytes(t, tc.doc)
			if res.Name != tc.want {
				t.Errorf("Name = %q, want %q", res.Name, tc.want)
			}
			if strings.ContainsRune(res.Name, '\ufeff') {
				t.Errorf("Name %q carries a BOM", res.Name)
			}
			if res.PointCount != 2 {
				t.Errorf("PointCount = %d, want 2", res.PointCount)
			}
		})
	}
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(b)
}

func TestSanitizer(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain ascii", "abc <x/>\n\r\t", "abc <x/>\n\r\t"},
		{"c0 controls become spaces", "a\x00b\x01c\x08d\x0be\x0cf\x1fg", "a b c d e f g"},
		{"del is kept", "a\x7fb", "a\x7fb"},
		{"multi-byte runes survive", "h\u00e9llo \u65e5\u672c \U0001F600", "h\u00e9llo \u65e5\u672c \U0001F600"},
		{"invalid byte", "a\xffb", "a\ufffdb"},
		{"lone continuation byte", "a\x80b", "a\ufffdb"},
		{"overlong encoding", "a\xc0\xafb", "a\ufffd\ufffdb"},
		{"surrogate encoded in utf-8", "a\xed\xa0\x80b", "a\ufffd\ufffd\ufffdb"},
		{"U+FFFE", "a\xef\xbf\xbeb", "a\ufffdb"},
		{"U+FFFF", "a\xef\xbf\xbfb", "a\ufffdb"},
		{"U+FFFD itself is kept", "a\ufffdb", "a\ufffdb"},
		{"dangling two-byte prefix at EOF", "ab\xc3", "ab\ufffd"},
		{"dangling three-byte prefix at EOF", "ab\xe6\x97", "ab\ufffd"},
		{"dangling four-byte prefix at EOF", "ab\xf0\x9f\x98", "ab\ufffd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := readAll(t, &sanitizer{src: strings.NewReader(tc.in)}); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			// A rune split across Read calls must come out as it would in one read.
			split := &sanitizer{src: iotest.OneByteReader(strings.NewReader(tc.in))}
			if got := readAll(t, split); got != tc.want {
				t.Errorf("one byte at a time: got %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("a multi-byte rune split across reads stays intact", func(t *testing.T) {
		in := strings.Repeat("\u65e5\u672c\U0001F600", 5000)
		got := readAll(t, &sanitizer{src: iotest.OneByteReader(strings.NewReader(in))})
		if got != in {
			t.Errorf("got %d bytes back, want the %d byte input unchanged", len(got), len(in))
		}
	})
	t.Run("the underlying error is propagated unchanged", func(t *testing.T) {
		s := &sanitizer{src: io.MultiReader(strings.NewReader("ab"), errReader{ErrTooLarge})}
		b, err := io.ReadAll(s)
		if err != ErrTooLarge {
			t.Fatalf("err = %v, want ErrTooLarge unchanged", err)
		}
		if string(b) != "ab" {
			t.Errorf("got %q before the error, want %q", b, "ab")
		}
	})
	t.Run("a partial rune before an error is flushed", func(t *testing.T) {
		s := &sanitizer{src: io.MultiReader(strings.NewReader("a\xe6"), errReader{ErrTooLarge})}
		b, err := io.ReadAll(s)
		if !errors.Is(err, ErrTooLarge) || string(b) != "a\ufffd" {
			t.Errorf("got (%q, %v)", b, err)
		}
	})
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }

func TestRunGuard(t *testing.T) {
	t.Run("run of exactly max passes, one more fails", func(t *testing.T) {
		const max = 8
		ok := "<a>" + strings.Repeat("x", max-2) + "<b>"  // run "a>" + 6 x = 8
		bad := "<a>" + strings.Repeat("x", max-1) + "<b>" // run of 9
		if got := readAll(t, &runGuard{r: strings.NewReader(ok), max: max}); got != ok {
			t.Errorf("got %q, want the input", got)
		}
		_, err := io.ReadAll(&runGuard{r: strings.NewReader(bad), max: max})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("err = %v, want ErrMalformed", err)
		}
	})
	t.Run("the run is counted across Read calls", func(t *testing.T) {
		in := "<a>" + strings.Repeat("x", 20)
		_, err := io.ReadAll(&runGuard{r: iotest.OneByteReader(strings.NewReader(in)), max: 10})
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("err = %v, want ErrMalformed", err)
		}
	})
	t.Run("a less-than sign resets the run", func(t *testing.T) {
		in := strings.Repeat("<abcdefg", 1000)
		if got := readAll(t, &runGuard{r: strings.NewReader(in), max: 8}); got != in {
			t.Errorf("input was altered")
		}
	})
	t.Run("the underlying error is propagated", func(t *testing.T) {
		g := &runGuard{r: io.MultiReader(strings.NewReader("<a>"), errReader{ErrTooLarge}), max: 8}
		if _, err := io.ReadAll(g); err != ErrTooLarge {
			t.Errorf("err = %v, want ErrTooLarge", err)
		}
	})

	descDoc := func(textLen int) string {
		// The run after "<desc" is "desc>" plus the text, up to the next '<'.
		return `<gpx><metadata><desc>` + strings.Repeat("d", textLen) + `</desc></metadata><trk>` +
			`<trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`
	}
	t.Run("through Parse, exactly maxRun parses", func(t *testing.T) {
		res, err := Parse(strings.NewReader(descDoc(maxRun-len("desc>"))), 25<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if len(res.Description) != maxRun-len("desc>") {
			t.Errorf("Description length %d", len(res.Description))
		}
	})
	t.Run("through Parse, maxRun+1 is malformed", func(t *testing.T) {
		res, err := Parse(strings.NewReader(descDoc(maxRun-len("desc>")+1)), 25<<20, "")
		if !errors.Is(err, ErrMalformed) || res != nil {
			t.Fatalf("got (non-nil result: %v, %v), want (nil, ErrMalformed)", res != nil, err)
		}
	})
}
