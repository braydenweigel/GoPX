package gpx

import (
	"encoding/xml"
	"os"
	"strconv"
	"time"
)

// WriteGPX writes the activity to a GPX 1.1 file at path. An existing file at path is overwritten.
// Heart rate is written as a Garmin TrackPointExtension so other apps (Strava, Garmin Connect) can read it.
// Calculated fields (distance, duration, elevation gain) aren't written. They're recalculated when the file is parsed.
func (g GPX) WriteGPX(path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	//report an error closing the file (e.g. the disk is full) unless something else failed first
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	if _, err = f.WriteString(xml.Header); err != nil {
		return err
	}

	enc := xml.NewEncoder(f)
	enc.Indent("", " ")
	if err = enc.Encode(g.toXML()); err != nil {
		return err
	}

	_, err = f.WriteString("\n")
	return err
}

// The types below control exactly what WriteGPX outputs. They're separate from the GPX models because
// encoding/xml can't write the namespace attributes or the "gpxtpx:" prefix from the models' tags,
// and writes small floats in exponent form (1e-05), which GPX doesn't allow.
// Names containing a colon (like "gpxtpx:hr") are written exactly as given.

type gpxXML struct {
	XMLName  xml.Name     `xml:"gpx"`
	Version  string       `xml:"version,attr"`
	Creator  string       `xml:"creator,attr"`
	Xmlns    string       `xml:"xmlns,attr"`
	XmlnsTPX string       `xml:"xmlns:gpxtpx,attr"`
	Metadata *metadataXML `xml:"metadata,omitempty"`
	Track    trackXML     `xml:"trk"`
}

type metadataXML struct {
	Time string `xml:"time"`
}

type trackXML struct {
	Name   string       `xml:"name,omitempty"`
	Type   string       `xml:"type,omitempty"`
	Points []trackPtXML `xml:"trkseg>trkpt"`
}

type trackPtXML struct {
	Lat        string         `xml:"lat,attr"`
	Lon        string         `xml:"lon,attr"`
	Elevation  string         `xml:"ele"`
	Time       string         `xml:"time,omitempty"`
	Extensions *extensionsXML `xml:"extensions,omitempty"`
}

type extensionsXML struct {
	HeartRate int `xml:"gpxtpx:TrackPointExtension>gpxtpx:hr"`
}

// toXML converts the activity into the types WriteGPX encodes.
func (g GPX) toXML() gpxXML {
	out := gpxXML{
		Version:  "1.1",
		Creator:  "GoPX",
		Xmlns:    "http://www.topografix.com/GPX/1/1",
		XmlnsTPX: "http://www.garmin.com/xmlschemas/TrackPointExtension/v1",
		Track: trackXML{
			Name:   g.Track.Name,
			Type:   g.Track.Type,
			Points: make([]trackPtXML, 0, len(g.Track.Segment.TrackPoints)),
		},
	}

	//metadata is optional. leave it out rather than writing a zero time (0001-01-01)
	if !g.StartTime.IsZero() {
		out.Metadata = &metadataXML{Time: formatTime(g.StartTime)}
	}

	for _, p := range g.Track.Segment.TrackPoints {
		pt := trackPtXML{
			Lat:       formatFloat(p.Lat),
			Lon:       formatFloat(p.Lon),
			Elevation: formatFloat(p.Elevation),
		}
		if !p.Time.IsZero() {
			pt.Time = formatTime(p.Time)
		}
		if p.HeartRate != nil {
			pt.Extensions = &extensionsXML{HeartRate: *p.HeartRate}
		}
		out.Track.Points = append(out.Track.Points, pt)
	}

	return out
}

// formatFloat writes a number in plain decimal form (never exponent form), using as few digits as needed.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatTime writes a time in the format GPX uses (e.g. 2026-03-15T16:00:00Z).
// Fractional seconds are included only when present, such as on points interpolated by a trim.
func formatTime(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
