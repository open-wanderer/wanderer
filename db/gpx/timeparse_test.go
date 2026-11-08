package gpx

import "testing"

// Expected values come from the TypeScript parser (parseGpxTime in
// web/src/lib/models/gpx/waypoint.ts) run once under node with TZ=UTC; the
// zone-less rows differ from TS only in that Go reads them as UTC, not local
// time (A3).
func TestParseTime(t *testing.T) {
	const noMatch = 0
	tests := []struct {
		name string
		in   string
		want int64
		ok   bool
	}{
		{"zulu", "2024-05-01T10:00:00Z", 1714557600000, true},
		{"lowercase zulu", "2024-05-01T10:00:00z", 1714557600000, true},
		{"fraction truncated to milliseconds", "2024-05-01T10:00:00.123456Z", 1714557600123, true},
		{"fraction padded to milliseconds", "2024-05-01T10:00:00.5Z", 1714557600500, true},
		{"comma fraction", "2024-05-01T10:00:00,5Z", 1714557600500, true},
		{"positive offset", "2024-05-01T12:00:00+02:00", 1714557600000, true},
		{"offset without colon", "2024-05-01T12:00:00+0200", 1714557600000, true},
		{"hour-only offset", "2024-05-01T10:00:00+02", 1714550400000, true},
		{"offset after a space", "2024-05-01T10:00:00 +0200", 1714550400000, true},
		{"zulu after a space", "2024-05-01T10:00:00 Z", 1714557600000, true},
		{"negative offset with minutes", "2024-05-01T10:00:00-00:30", 1714559400000, true},
		{"basic format", "20240501T100000Z", 1714557600000, true},
		{"space separator", "2024-05-01 10:00:00Z", 1714557600000, true},
		{"date only is UTC midnight", "2024-05-01", 1714521600000, true},
		{"zone-less is UTC", "2024-05-01T10:00:00", 1714557600000, true},
		{"hour only", "2024-05-01T10", 1714557600000, true},
		{"hour and minute", "2024-05-01T10:30", 1714559400000, true},
		{"surrounding whitespace", "  2024-05-01T10:00:00Z \n", 1714557600000, true},
		{"epoch", "1970-01-01T00:00:00Z", 0, true},
		{"one millisecond before the epoch", "1969-12-31T23:59:59.999Z", -1, true},
		{"year zero", "0000-01-01", -62167219200000, true},
		{"hour 24 is the next midnight", "2024-12-31T24:00:00Z", 1735689600000, true},
		{"hour 24 with a zero fraction", "2024-05-01T24:00:00.000Z", 1714608000000, true},
		{"hour 24 with an offset", "2024-05-01T24:00:00+01:00", 1714604400000, true},
		{"signed six-digit year", "+012345-01-01T00:00:00Z", 327403382400000, true},
		{"negative six-digit year", "-000001-01-01T00:00:00Z", -62198755200000, true},

		// V8 normalises a day that the month lacks instead of rejecting it, so
		// the TypeScript oracle reads Feb 30 as Mar 1 (this refines the plan's
		// A8, which assumed rejection).
		{"Feb 30 rolls over", "2024-02-30", 1709251200000, true},
		{"Feb 30 with time rolls over", "2024-02-30T10:00:00Z", 1709287200000, true},
		{"Feb 29 in a common year rolls over", "2023-02-29", 1677628800000, true},
		{"Apr 31 rolls over", "2024-04-31T10:00:00Z", 1714557600000, true},

		{"not a time", "N/A", noMatch, false},
		{"empty", "", noMatch, false},
		{"whitespace only", "   ", noMatch, false},
		{"month 13", "2024-13-01T10:00:00Z", noMatch, false},
		{"month 0", "2024-00-10", noMatch, false},
		{"day 0", "2024-05-00", noMatch, false},
		{"day 32", "2024-01-32", noMatch, false},
		{"hour 25", "2024-05-01T25:00:00Z", noMatch, false},
		{"hour 24 with minutes", "2024-05-01T24:01:00Z", noMatch, false},
		{"hour 24 with seconds", "2024-05-01T24:00:01Z", noMatch, false},
		{"hour 24 with fraction", "2024-05-01T24:00:00.500Z", noMatch, false},
		{"minute 60", "2024-05-01T10:60:00Z", noMatch, false},
		{"second 60", "2024-05-01T10:00:60Z", noMatch, false},
		{"offset hour 24", "2024-05-01T10:00:00+24:00", noMatch, false},
		{"offset minute 60", "2024-05-01T10:00:00+02:60", noMatch, false},
		{"dot with no fraction digits", "2024-05-01T10:00:00.Z", noMatch, false},
		{"single-digit minute", "2024-05-01T10:0", noMatch, false},
		{"month name", "Jan 1 2024", noMatch, false},

		// The grammar admits signed and 5 to 6 digit years that V8 reads through
		// its legacy parser with quirky or invalid results ("-0001-01-01" is
		// 2001, "000000-01-01" is 2000, "+2024-05-01T10:00:00Z" is invalid).
		// No exporter writes them, so Go reads them as "no time" rather than
		// copy the quirks. Only an unsigned four-digit year and a signed
		// six-digit year, which are valid ISO 8601 and read the same everywhere,
		// are times.
		{"signed four-digit year", "+2024-05-01T10:00:00Z", noMatch, false},
		{"signed four-digit date", "+2024-05-01", noMatch, false},
		{"five-digit year", "12345-01-01", noMatch, false},
		{"unsigned six-digit year", "000000-01-01", noMatch, false},
		{"negative zero year", "-000000-01-01T00:00:00Z", noMatch, false},
		{"year past the Date range", "+275761-01-01T00:00:00Z", noMatch, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseTime(tc.in)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Errorf("parseTime(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}
