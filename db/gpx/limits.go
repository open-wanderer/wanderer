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
	// maxTextBytes is the total retained name, desc, sym and type text.
	maxTextBytes = 8 << 20
)
