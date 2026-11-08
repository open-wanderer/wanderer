package gpx

// maxRun is the longest run of bytes without a '<' the reader lets through.
// encoding/xml buffers a whole name, attribute or text run before it returns
// it, so this bounds that buffer. 1 MiB is far beyond any real description and
// keeps one hostile run small next to the byte cap.
const maxRun = 1 << 20
