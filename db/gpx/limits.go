package gpx

// maxRun is the longest run of bytes without a '<' the reader lets through.
// encoding/xml buffers a whole name, attribute or text run before it returns
// it, so this bounds that buffer. 1 MiB is far beyond any real description and
// keeps one hostile run small next to the byte cap.
const maxRun = 1 << 20

// Structural limits. Each bounds one way a hostile document could make the
// parser hold memory the byte cap alone does not bound, and each is an
// internal constant rather than configuration. A document over one of them
// fails with a typed error; none is a safe truncation.
const (
	// maxDepth is the deepest element nesting. encoding/xml allocates per open
	// element (about 30 bytes of input cost 90 bytes of heap), and real GPX is
	// under 8 deep, so a quarter of this is already generous.
	maxDepth = 32
	// maxPoints is the stored track and route points together. 500,000 points
	// take about 16 MB in the struct-of-arrays store and are far past the point
	// count of the longest real recordings (multi-day tracks are tens of
	// thousands of points).
	maxPoints = 500_000
	// maxSegments is the stored, non-empty segments. A segment costs a slice
	// header set, so it is capped by count rather than by points.
	maxSegments = 10_000
	// maxWaypoints is the stored waypoints, each of which keeps its text.
	maxWaypoints = 50_000
	// maxTracks is the root-level <trk> and <rte> elements together. Every one
	// costs a stored measuredTrack whether or not it ever holds a point, and an
	// empty one cannot be told apart from a populated one when it starts, so
	// each one counts. A file past maxSegments non-empty tracks already fails
	// that limit, so this only rejects documents made of empty elements.
	maxTracks = 10_000
	// maxTextBytes is the total retained name, desc, sym and type text.
	maxTextBytes = 8 << 20
	// maxNumberText bounds the text collected for one <ele>, <time> or <number>
	// element and the length parseNumber accepts. A longer value is unusable,
	// which is not an error: numeric fields are never retained, so this only
	// bounds the per-element buffer. No real value is near 256 bytes. The cap
	// counts the raw collected bytes, including leading and trailing whitespace,
	// before parseNumber trims them, so an <ele>, <time> or <number> padded past
	// 256 bytes is unusable (a missing optional value, not an error) where TS and
	// Dart would read the trimmed value. No real exporter pads that far.
	maxNumberText = 256
)

// Value bounds. A number beyond one of these is unusable, not an error, exactly
// like text that is not a number: a coordinate beyond the bound drops its point
// and an elevation beyond it is no elevation. The bounds are far outside any
// real value (coordinates sit within +-180, and corpus fixture 16 deliberately
// keeps 95.5 and 190). They exist so that 500,000 stored points can overflow
// neither a float64 sum nor the polyline's integer encoding: at most 5e11 for a
// centroid sum, 1e12 for elevation gain or loss, and 1e11 for round(v*1e5).
const (
	// maxAbsCoordinate is the largest latitude or longitude magnitude kept, in degrees.
	maxAbsCoordinate = 1e6
	// maxAbsElevation is the largest elevation magnitude kept, in metres.
	maxAbsElevation = 1e6
)
