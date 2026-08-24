package polyline

import "testing"

func TestDecode(t *testing.T) {
	points, err := Decode("_p~iF~ps|U_ulLnnqC_mqNvxq`@", 1e5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}
	if points[0][0] != 38.5 || points[0][1] != -120.2 {
		t.Fatalf("unexpected first point: %#v", points[0])
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	input := [][2]float64{{38.5, -120.2}, {40.7, -120.95}, {43.252, -126.453}}
	encoded, err := Encode(input, 1e5)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded != "_p~iF~ps|U_ulLnnqC_mqNvxq`@" {
		t.Fatalf("encoded polyline = %q", encoded)
	}
	decoded, err := Decode(encoded, 1e5)
	if err != nil {
		t.Fatalf("decode encoded polyline: %v", err)
	}
	if len(decoded) != len(input) || decoded[2] != input[2] {
		t.Fatalf("round trip = %#v", decoded)
	}
}

func TestEncodeRejectsZeroPrecision(t *testing.T) {
	if _, err := Encode([][2]float64{{1, 2}}, 0); err == nil {
		t.Fatal("zero precision was accepted")
	}
}

func TestNormalizeCoordinateScale(t *testing.T) {
	points := [][2]float64{{385, -1202}}
	NormalizeCoordinateScale(points)
	if points[0][0] != 38.5 || points[0][1] != -120.2 {
		t.Fatalf("expected normalized point, got %#v", points[0])
	}
}

func TestShouldSwapCoordinates(t *testing.T) {
	coords := [][2]float64{{120.2, 38.5}, {121.0, 39.0}}
	if !ShouldSwapCoordinates(coords) {
		t.Fatal("expected coordinates to be detected as swapped")
	}
}

func TestProportionalIndex(t *testing.T) {
	if got := ProportionalIndex(2, 5, 3); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := ProportionalIndex(4, 5, 3); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}
