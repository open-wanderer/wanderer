package gpx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"
)

type Point struct {
	Lat       float64
	Lon       float64
	Elevation *float64
	Time      *time.Time
}

func Track(creator string, name string, points []Point) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("track has no points")
	}
	if creator == "" {
		creator = "wanderer plugin"
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<gpx version="1.1" creator="`)
	_ = xml.EscapeText(&buf, []byte(creator))
	buf.WriteString(`" xmlns="http://www.topografix.com/GPX/1/1">`)
	buf.WriteString("<trk>")
	buf.WriteString("<name>")
	_ = xml.EscapeText(&buf, []byte(name))
	buf.WriteString("</name>")
	buf.WriteString("<trkseg>")
	for _, point := range points {
		buf.WriteString(`<trkpt lat="`)
		buf.WriteString(strconv.FormatFloat(point.Lat, 'f', 8, 64))
		buf.WriteString(`" lon="`)
		buf.WriteString(strconv.FormatFloat(point.Lon, 'f', 8, 64))
		buf.WriteString(`">`)
		if point.Elevation != nil {
			buf.WriteString("<ele>")
			buf.WriteString(strconv.FormatFloat(*point.Elevation, 'f', 2, 64))
			buf.WriteString("</ele>")
		}
		if point.Time != nil {
			buf.WriteString("<time>")
			buf.WriteString(point.Time.UTC().Format(time.RFC3339))
			buf.WriteString("</time>")
		}
		buf.WriteString("</trkpt>")
	}
	buf.WriteString("</trkseg>")
	buf.WriteString("</trk>")
	buf.WriteString("</gpx>")
	return buf.Bytes(), nil
}
