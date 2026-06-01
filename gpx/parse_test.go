package gpx

import (
	"testing"
	"time"
)

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

	//Test GPX Track Total Time
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
