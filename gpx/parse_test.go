package gpx

import (
	"errors"
	"io/fs"
	"math"
	"testing"
	"time"
)

// Tests ParseFile() to verify parsing and stat calculations work properly
func TestParseFile(t *testing.T) {
	file := "testdata/test1.gpx"

	gpx, err := ParseFile(file)
	if err != nil {
		t.Error("Error: ", err.Error())
	}

	//Test GPX Track Name
	if gpx.Track.Name != "Shamrock Run 8k" {
		t.Error("Error: Error parsing track name")
	}

	//Test GPX Metadata Time
	expected, err := time.Parse(time.RFC3339, "2026-03-15T16:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	if !gpx.StartTime.Equal(expected) {
		t.Errorf("Error: Expected %v, got %v", expected, gpx.StartTime)
	}

	//Test GPX Track Type
	if gpx.Track.Type != "running" {
		t.Error("Error: Error parsing track type")
	}

	//Test GPX Track Total Distance
	if gpx.Track.Distance != 8054.251460047391 {
		t.Errorf("Error: Expected %v, got %v", 8054.251460047391, gpx.Track.Distance)
	}

	//Test GPX Track Activity Time
	if gpx.Track.ActivityTime.Seconds() != 1644 {
		t.Errorf("Error: Expected %v, got %v", 1644, gpx.Track.ActivityTime.Seconds())
	}

	//Test GPX Track Elevation Gain
	if gpx.Track.ElevationGain != 13.3 {
		t.Errorf("Error: Expected %v, got %v", 13.3, gpx.Track.ElevationGain)
	}

	//Test GPX Track Net Elevation Change (9.8m at the end - 9.5m at the start)
	if math.Abs(gpx.Track.NetElev-0.3) > 1e-9 {
		t.Errorf("Error: Expected %v, got %v", 0.3, gpx.Track.NetElev)
	}

	//Test GPX Track Average Heart Rate
	if gpx.Track.HeartRate == nil || *gpx.Track.HeartRate != 175 {
		t.Errorf("Error: Expected %v, got %v", 175, gpx.Track.HeartRate)
	}

	//Test GPX Track Segment Length
	if len(gpx.Track.Segment.TrackPoints) != 1644 {
		t.Errorf("Error: Expected %v, got %v", 1644, len(gpx.Track.Segment.TrackPoints))
	}
}

// Tests ParseFile() to verify that an empty <trkseg> returns an error
func TestParseFileEmptySegment(t *testing.T) {
	file := "testdata/test2.gpx"

	_, err := ParseFile(file)
	if err.Error() != "No Data" {
		t.Errorf("Error: Expected %v, got %v", "No Data", err.Error())
	}
}

// Tests ParseFile() with a GPX file with a single trackpoint
func TestParseFileSinglePoint(t *testing.T) {
	file := "testdata/test3.gpx"

	gpx, err := ParseFile(file)
	if err != nil {
		t.Errorf("Error: %v", err.Error())
	}

	//Test GPX Track Total Distance
	if gpx.Track.Distance != 0 {
		t.Errorf("Error: Expected %v, got %v", 0, gpx.Track.Distance)
	}

	//Test GPX Track Activity Time
	if gpx.Track.ActivityTime.Seconds() != 0 {
		t.Errorf("Error: Expected %v, got %v", 0, gpx.Track.ActivityTime.Seconds())
	}

	//Test GPX Track Elevation Gain
	if gpx.Track.ElevationGain != 0 {
		t.Errorf("Error: Expected %v, got %v", 0, gpx.Track.ElevationGain)
	}

	//Test GPX Track Segment Length
	if len(gpx.Track.Segment.TrackPoints) != 1 {
		t.Errorf("Error: Expected %v, got %v", 1, len(gpx.Track.Segment.TrackPoints))
	}

}

// Tests ParseFile() to verify that an invalid file format returns an error
func TestParseFileInvalidFile(t *testing.T) {
	file := "testdata/test4.gpx"

	_, err := ParseFile(file)
	if err == nil || err.Error() != "expected element type <gpx> but have <goop>" {
		t.Errorf("expected XML element error, got: %v", err)
	}
}

// Tests ParseFile() to verify that a non-existent file returns an error
func TestParseFileNoFile(t *testing.T) {
	file := "testdata/file_that_does_not_exist.gpx"

	_, err := ParseFile(file)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected file-not-found error, got: %v", err)
	}
}

// Tests calcStats() to verify the average heart rate is weighted by time, not by number of points
func TestCalcStatsHeartRate(t *testing.T) {
	start := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)
	track := Track{Segment: TrackSeg{TrackPoints: []TrackPoint{
		{Time: start, HeartRate: intPtr(100)},
		{Time: start.Add(100 * time.Second), HeartRate: intPtr(100)},
		{Time: start.Add(400 * time.Second), HeartRate: intPtr(200)},
	}}}

	if err := calcStats(&track); err != nil {
		t.Fatal(err)
	}

	//100bpm for 100s, then 100->200bpm (avg 150) for 300s: (100*100 + 150*300) / 400 = 137.5
	//a plain average of the points would be (100+100+200) / 3 = 133
	if track.HeartRate == nil || *track.HeartRate != 138 {
		t.Errorf("Error: Expected heart rate %v, got %v", 138, track.HeartRate)
	}
}

// Tests calcStats() to verify the average heart rate is nil without heart rate data, and uses the reading for a single point
func TestCalcStatsHeartRateEdgeCases(t *testing.T) {
	start := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)

	track := Track{Segment: TrackSeg{TrackPoints: []TrackPoint{
		{Time: start},
		{Time: start.Add(100 * time.Second)},
	}}}
	if err := calcStats(&track); err != nil {
		t.Fatal(err)
	}
	if track.HeartRate != nil {
		t.Errorf("Error: Expected nil heart rate, got %v", *track.HeartRate)
	}

	track = Track{Segment: TrackSeg{TrackPoints: []TrackPoint{
		{Time: start, HeartRate: intPtr(150)},
	}}}
	if err := calcStats(&track); err != nil {
		t.Fatal(err)
	}
	if track.HeartRate == nil || *track.HeartRate != 150 {
		t.Errorf("Error: Expected heart rate %v, got %v", 150, track.HeartRate)
	}
}
