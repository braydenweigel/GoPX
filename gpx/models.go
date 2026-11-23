package gpx

import (
	"encoding/xml"
	"time"
)

// GPX is a parsed GPX activity: a single track recorded by a GPS device or app.
// Create one with [ParseFile]. Fields tagged xml:"-" are calculated when the file is
// parsed (or trimmed) and aren't part of the GPX file itself.
type GPX struct {
	XMLName   xml.Name  `xml:"gpx"`
	Track     Track     `xml:"trk"`
	StartTime time.Time `xml:"metadata>time"` //the file's <metadata><time>. zero if the file has none
}

// Track is the recorded activity: its points and the totals calculated from them.
type Track struct {
	Name    string   `xml:"name"`
	Type    string   `xml:"type"` //activity type as written by the recording app, e.g. "running"
	Segment TrackSeg `xml:"trkseg"`

	// Calculated fields
	Distance      float64       `xml:"-"` //total distance in meters
	ActivityTime  time.Duration `xml:"-"` //time from the first point to the last, including stops
	ElevationGain float64       `xml:"-"` //total elevation climbed in meters. descents don't subtract from it
	NetElev       float64       `xml:"-"` //elevation at the last point minus elevation at the first, in meters
	HeartRate     *int          `xml:"-"` //time-weighted average heart rate. nil if the activity has no heart rate data
}

// TrackSeg holds a track's points in the order they were recorded.
type TrackSeg struct {
	TrackPoints []TrackPoint `xml:"trkpt"`
}

// TrackPoint is a single recorded GPS point. The calculated fields are running totals
// from the start of the activity, so the last point's values equal the track totals.
type TrackPoint struct {
	// XML fields
	Lat       float64   `xml:"lat,attr"`
	Lon       float64   `xml:"lon,attr"`
	Elevation float64   `xml:"ele"`                               //meters
	Time      time.Time `xml:"time"`                              //zero if the point has no time
	HeartRate *int      `xml:"extensions>TrackPointExtension>hr"` //beats per minute from the Garmin extension. nil if not recorded

	// Calculated fields
	Distance float64       `xml:"-"` //meters from the start of the activity
	Duration time.Duration `xml:"-"` //time since the first point
	ElevGain float64       `xml:"-"` //elevation climbed since the start of the activity, in meters
}

// Coordinate is a latitude and longitude in decimal degrees.
type Coordinate struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
}

// DistanceSplits is the result of [GPX.CalcSplitsDistance]: the inputs used and the splits calculated.
type DistanceSplits struct {
	Units   Unit    //the units (miles, kilometers, feet) of Start, Length, and each Split.Length
	Start   float64 //the distance the first split starts from. 0 starts at the beginning of the activity
	Length  float64 //the length of each split
	Partial bool    //true includes a partial split at the end of the activity. A 4.5km activity can return Splits with lengths of 1, 1, 1, 1, and 0.5. Only applies when splitting an entire activity
	Splits  []Split
}

// TimeSplits is the result of [GPX.CalcSplitsTime]: the inputs used and the splits calculated.
type TimeSplits struct {
	Start   float64 //the time in seconds the first split starts from. 0 starts at the beginning of the activity. 120 starts from 2 minutes
	Length  float64 //the length of each split in seconds
	Partial bool    //true includes a partial split at the end of the activity. A 2000 second activity can return Splits with Durations of 600, 600, 600, and 200. Only applies when splitting an entire activity
	Splits  []Split
}

// Unit is a unit of distance.
type Unit string

// Supported units of distance.
const (
	Miles      Unit = "mi"
	Kilometers Unit = "km"
	Feet       Unit = "ft"
)

// Split is one section of an activity, from [GPX.CalcSplitsDistance] or [GPX.CalcSplitsTime].
// The start and end of a split rarely land exactly on a recorded point, so values there are
// interpolated between the points on either side.
type Split struct {
	Length     float64       //distance covered. for distance splits: in DistanceSplits.Units, matching DistanceSplits.Length except for a partial last split. for time splits: in meters
	Duration   time.Duration //duration of the split
	StartPoint Coordinate    //coordinates at the beginning of the split
	EndPoint   Coordinate    //coordinates at the end of the split
	ElevGain   float64       //elevation climbed during the split, in meters
	NetElev    float64       //elevation at the end minus elevation at the start, in meters
	HeartRate  *int          //time-weighted average heart rate of the split. nil if there's no heart rate data in the split
}
