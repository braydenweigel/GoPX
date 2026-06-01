package gpx

import (
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
	if err.Error() != "expected element type <gpx> but have <goop>" {
		t.Error("Error: Expected Error")
	}
}

// Tests ParseFile() to verify that a non-existent file returns an error
func TestParseFileNoFile(t *testing.T) {
	file := "testdata/file_that_does_not_exist.gpx"

	_, err := ParseFile(file)
	if err.Error() != "open testdata/file_that_does_not_exist.gpx: no such file or directory" {
		t.Error("Error: Expected Error")
	}
}
