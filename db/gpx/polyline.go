package gpx

// rdpWorkBudget is the most points the simplification may scan before
// Polyline gives up with ErrTooComplex (A2).
var rdpWorkBudget int64 = 2_000_000_000

// Polyline is the encoded, simplified track. Stub for the RED step.
func (r *Result) Polyline() (string, error) { return "", nil }
