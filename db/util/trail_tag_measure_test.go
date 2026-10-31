package util

import "testing"

func TestParseTagMeasure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    float64
		wantOK  bool
	}{
		{name: "decimal with unit", content: "12.5m", want: 12.5, wantOK: true},
		{name: "wire format", content: "100.000000m", want: 100, wantOK: true},
		{name: "negative", content: "-3.5m", want: -3.5, wantOK: true},
		{name: "empty", content: ""},
		{name: "unit only", content: "m"},
		{name: "not a number, one character", content: "x"},
		{name: "not a number", content: "abc"},
		{name: "NaN", content: "NaNm"},
		{name: "positive infinity", content: "Infm"},
		{name: "negative infinity", content: "-Infm"},
		{name: "out of float64 range", content: "1e999m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseTagMeasure(tc.content)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("parseTagMeasure(%q) = (%v, %v), want (%v, %v)", tc.content, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
