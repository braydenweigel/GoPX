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
	Units   Unit    //the units (miles, kilometers, meters) of the split
	Start   float64 //the point the first split starts from. should match units used. use 0 to start at beginning of activity
	Length  float64 //the length of the split
	Partial bool    //"true" includes partials splits at end of activity. A 4.5km activity can return Splits with Distances of 1, 1, 1, 1, and 0.5 <-- Omitted when false. Only matters when splitting an entire activity
	Splits  []Split
}

// Unit is a unit of distance.
type Unit string

const (
	Miles      Unit = "mi"
	Kilometers Unit = "km"
	Feet       Unit = "ft"
)

type Split struct {
	Length     float64       //length of the split. matches DistanceSplits.Length except for the last Split when Partial == true
	Duration   time.Duration //duration of the split
	StartPoint Coordinate    //coordinates at the beginning of the split
	EndPoint   Coordinate    //coordinates at the end of the split
	ElevGain   float64       //elevation gain of the split
	HeartRate  *int          //average heart rate of the split
}
