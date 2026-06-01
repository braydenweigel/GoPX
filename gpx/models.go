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
	TotalDistance float64       `xml:"-"`
	TotalTime     time.Duration `xml:"-"`
	TotalElevGain float64       `xml:"-"`
}
