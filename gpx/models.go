package gpx

import (
	"encoding/xml"
	"time"
)

type GPX struct {
	XMLName   xml.Name  `xml:"gpx"`
	Track     Track     `xml:"trk"`
	StartTime time.Time `xml:"metadata>time"`
}

type Track struct {
	Name          string        `xml:"name"`
	Type          string        `xml:"type"`
	Segment       TrackSeg      `xml:"trkseg"`
	Distance      float64       `xml:"-"`
	ActivityTime  time.Duration `xml:"-"`
	ElevationGain float64       `xml:"-"`
}

type TrackSeg struct {
	TrackPoints []TrackPoint `xml:"trkpt"`
}

type TrackPoint struct {
	// XML fields
	Lat       float64   `xml:"lat,attr"`
	Lon       float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`
	Time      time.Time `xml:"time"`
	HeartRate *int      `xml:"extensions>TrackPointExtension>hr"`

	// Calculated fields
	Distance float64       `xml:"-"`
	Duration time.Duration `xml:"-"`
	ElevGain float64       `xml:"-"`
}

type Coordinate struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
}

type DistanceSplits struct {
	Units    string
	Start    float64
	Distance float64
	Partial  bool //"true" includes partials splits at end of activity. A 4.5km activity can return Splits with Distances of 1, 1, 1, 1, and 0.5 <-- Omitted when false
	Splits   []Split
}

// Unit is a unit of distance.
type Unit string

const (
	Miles      Unit = "mi"
	Kilometers Unit = "km"
	Feet       Unit = "ft"
)

type Split struct {
	Distance   float64
	Duration   time.Duration
	StartPoint Coordinate
	EndPoint   Coordinate
	ElevGain   float64
	HeartRate  *int
}
