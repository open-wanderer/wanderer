package gpx

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// timeGrammar is DART_DATETIME_GRAMMAR, transcribed verbatim from
// web/src/lib/models/gpx/waypoint.ts. RE2's \d is ASCII-only, like JavaScript's.
//
// Groups: 1 year, 2 month, 3 day, 4 hour, 5 minute, 6 second, 7 fraction,
// 8 zone, 9 offset sign, 10 offset hour, 11 offset minute.
var timeGrammar = regexp.MustCompile(`^([+-]?\d{4,6})-?(\d\d)-?(\d\d)(?:[ T](\d\d)(?::?(\d\d)(?::?(\d\d)(?:[.,](\d+))?)?)?( ?[zZ]| ?([-+])(\d\d)(?::?(\d\d))?)?)?$`)

// isoMillis is the layout of Date.prototype.toISOString for years 0000-9999,
// always with milliseconds and a literal Z. appendISOMillis handles the rest.
const isoMillis = "2006-01-02T15:04:05.000Z"

// maxDateMs is the range of a JavaScript Date, 275760-09-13T00:00:00Z.
const maxDateMs = 8_640_000_000_000_000

// parseTime reads a <time> body as Unix milliseconds. Anything that is not a
// well-formed ISO 8601 instant, in the grammar above, is "no time".
//
// It mirrors parseGpxTime and normalizedIsoString in waypoint.ts, which hand
// the normalised string to V8's Date: the fraction is truncated to three
// digits, minutes and seconds default to 00, and the basic format, a space
// separator, a comma fraction and an hour-only offset are all read. V8 rejects
// a month outside 1-12, a day outside 1-31, an hour above 24 (24 only as
// 24:00:00), a minute or second above 59, and an offset beyond 23:59; it
// accepts a day the month lacks and rolls it over (Feb 30 is Mar 1), which
// time.Date does too. A value without a zone is UTC in Go, local time in TS and
// Dart (A3); a date without a time is UTC midnight in all three.
//
// Only an unsigned four-digit year and a signed six-digit year (not -000000)
// are read. V8 reads the other forms the grammar admits through its legacy
// parser with invalid or quirky results, and no exporter writes them.
func parseTime(raw string) (int64, bool) {
	m := timeGrammar.FindStringSubmatch(strings.TrimFunc(raw, isTrimSpace))
	if m == nil {
		return 0, false
	}
	year, ok := parseYear(m[1])
	if !ok {
		return 0, false
	}
	month, day := atoi(m[2]), atoi(m[3])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, false
	}

	var hour, minute, second, millis, offsetMinutes int
	if m[4] != "" {
		hour = atoi(m[4])
		if m[5] != "" {
			minute = atoi(m[5])
		}
		if m[6] != "" {
			second = atoi(m[6])
		}
		if f := m[7]; f != "" {
			if len(f) > 3 {
				f = f[:3]
			}
			millis = atoi(f + strings.Repeat("0", 3-len(f)))
		}
		if hour > 24 || minute > 59 || second > 59 {
			return 0, false
		}
		if hour == 24 && (minute != 0 || second != 0 || millis != 0) {
			return 0, false
		}
		if m[9] != "" {
			zoneHour := atoi(m[10])
			zoneMinute := 0
			if m[11] != "" {
				zoneMinute = atoi(m[11])
			}
			if zoneHour > 23 || zoneMinute > 59 {
				return 0, false
			}
			offsetMinutes = zoneHour*60 + zoneMinute
			if m[9] == "-" {
				offsetMinutes = -offsetMinutes
			}
		}
	}

	local := time.Date(year, time.Month(month), day, hour, minute, second, millis*int(time.Millisecond), time.UTC)
	ms := local.UnixMilli() - int64(offsetMinutes)*60_000
	if ms > maxDateMs || ms < -maxDateMs {
		return 0, false
	}
	return ms, true
}

// appendISOMillis appends ms to dst in the exact form of
// Date.prototype.toISOString, and is the inverse of parseTime: parseTime reads
// every string it produces back to the same millisecond.
//
// ECMAScript writes years 0000-9999 as four digits and every other year as a
// sign plus six digits (+012345, -000001), which is the "expanded year" form.
// The in-range case uses isoMillis, and the expanded case writes the date and
// time fields by hand because a layout string cannot start with a sign.
func appendISOMillis(dst []byte, ms int64) []byte {
	t := time.UnixMilli(ms).UTC()
	year := t.Year()
	if year >= 0 && year <= 9999 {
		return t.AppendFormat(dst, isoMillis)
	}
	if year < 0 {
		dst = append(dst, '-')
		year = -year
	} else {
		dst = append(dst, '+')
	}
	dst = appendPadded(dst, year, 6)
	dst = append(dst, '-')
	dst = appendPadded(dst, int(t.Month()), 2)
	dst = append(dst, '-')
	dst = appendPadded(dst, t.Day(), 2)
	dst = append(dst, 'T')
	dst = appendPadded(dst, t.Hour(), 2)
	dst = append(dst, ':')
	dst = appendPadded(dst, t.Minute(), 2)
	dst = append(dst, ':')
	dst = appendPadded(dst, t.Second(), 2)
	dst = append(dst, '.')
	dst = appendPadded(dst, t.Nanosecond()/int(time.Millisecond), 3)
	return append(dst, 'Z')
}

// appendPadded appends the non-negative n as decimal digits, zero-padded to width.
func appendPadded(dst []byte, n, width int) []byte {
	digits := strconv.Itoa(n)
	for i := len(digits); i < width; i++ {
		dst = append(dst, '0')
	}
	return append(dst, digits...)
}

// parseYear accepts an unsigned four-digit year or a signed six-digit year.
func parseYear(s string) (int, bool) {
	signed := s[0] == '+' || s[0] == '-'
	digits := s
	if signed {
		digits = s[1:]
	}
	want := 4
	if signed {
		want = 6
	}
	if len(digits) != want {
		return 0, false
	}
	y := atoi(digits)
	if s[0] == '-' {
		if y == 0 {
			return 0, false
		}
		y = -y
	}
	return y, true
}

// atoi converts a run of ASCII digits the grammar has already vetted.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// isTrimSpace is JavaScript's String.prototype.trim set, which adds U+FEFF to
// the Unicode spaces.
func isTrimSpace(r rune) bool { return unicode.IsSpace(r) || r == 0xFEFF }
