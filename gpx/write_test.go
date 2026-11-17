package gpx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tests WriteGPX() to verify a parsed file can be written and parsed back with the same data
func TestWriteGPXRoundTrip(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "out.gpx")
	if err := g.WriteGPX(path); err != nil {
		t.Fatal(err)
	}

	written, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if written.Track.Name != g.Track.Name || written.Track.Type != g.Track.Type {
		t.Errorf("Error: Expected name %q and type %q, got %q and %q", g.Track.Name, g.Track.Type, written.Track.Name, written.Track.Type)
	}
	if !written.StartTime.Equal(g.StartTime) {
		t.Errorf("Error: Expected start time %v, got %v", g.StartTime, written.StartTime)
	}
	if len(written.Track.Segment.TrackPoints) != len(g.Track.Segment.TrackPoints) {
		t.Fatalf("Error: Expected %v points, got %v", len(g.Track.Segment.TrackPoints), len(written.Track.Segment.TrackPoints))
	}
	if written.Track.Distance != g.Track.Distance {
		t.Errorf("Error: Expected distance %v, got %v", g.Track.Distance, written.Track.Distance)
	}
	if written.Track.ActivityTime != g.Track.ActivityTime {
		t.Errorf("Error: Expected activity time %v, got %v", g.Track.ActivityTime, written.Track.ActivityTime)
	}
	if written.Track.ElevationGain != g.Track.ElevationGain {
		t.Errorf("Error: Expected elevation gain %v, got %v", g.Track.ElevationGain, written.Track.ElevationGain)
	}

	for i, p := range g.Track.Segment.TrackPoints {
		w := written.Track.Segment.TrackPoints[i]
		if w.Lat != p.Lat || w.Lon != p.Lon || w.Elevation != p.Elevation || !w.Time.Equal(p.Time) {
			t.Fatalf("Error: Point %d: Expected %+v, got %+v", i, p, w)
		}
		if (w.HeartRate == nil) != (p.HeartRate == nil) || (p.HeartRate != nil && *w.HeartRate != *p.HeartRate) {
			t.Fatalf("Error: Point %d: Expected heart rate %v, got %v", i, p.HeartRate, w.HeartRate)
		}
	}
}

// Tests WriteGPX() to verify the file has the GPX header, namespaces, and the Garmin heart rate extension
func TestWriteGPXFormat(t *testing.T) {
	start := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)
	g := gpxFromPoints([]TrackPoint{
		{Lat: 0.00001, Lon: -122.67087, Elevation: 9.5, Time: start, HeartRate: intPtr(171)},
		{Lat: 0.00002, Lon: -122.67087, Elevation: 10, Time: start.Add(1500 * time.Millisecond)},
	})
	g.StartTime = start
	g.Track.Name = "Test Run"

	path := filepath.Join(t.TempDir(), "out.gpx")
	if err := g.WriteGPX(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)

	expected := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`version="1.1"`,
		`xmlns="http://www.topografix.com/GPX/1/1"`,
		`xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v1"`,
		`<time>2026-03-15T16:00:00Z</time>`,
		`<name>Test Run</name>`,
		`lat="0.00001"`, //not 1e-05
		`<gpxtpx:TrackPointExtension>`,
		`<gpxtpx:hr>171</gpxtpx:hr>`,
		`<time>2026-03-15T16:00:01.5Z</time>`, //fractional seconds kept
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("Error: Expected output to contain %s\n%s", s, out)
		}
	}

	if strings.Count(out, "<extensions>") != 1 { //only the point with heart rate gets extensions
		t.Errorf("Error: Expected 1 <extensions>, got %v", strings.Count(out, "<extensions>"))
	}
	if strings.Contains(out, "<type>") { //empty type is left out
		t.Error("Error: Expected no <type> element")
	}
}

// Tests WriteGPX() to verify metadata is left out when there's no start time
func TestWriteGPXNoStartTime(t *testing.T) {
	g := makeTrack(500, nil)

	path := filepath.Join(t.TempDir(), "out.gpx")
	if err := g.WriteGPX(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "<metadata>") || strings.Contains(string(data), "0001-01-01") {
		t.Errorf("Error: Expected no metadata or zero times\n%s", data)
	}
}

// Tests WriteGPX() to verify a trimmed activity can be written and parsed back
func TestWriteGPXTrimmed(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx")
	if err != nil {
		t.Fatal(err)
	}

	trimmed, err := g.TrimDistance(Kilometers, 0.5, 0.5)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "trimmed.gpx")
	if err := trimmed.WriteGPX(path); err != nil {
		t.Fatal(err)
	}

	written, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	//distance is recalculated from coordinates when parsed, so allow a small difference from the interpolated cut points
	if diff := written.Track.Distance - trimmed.Track.Distance; diff > 0.5 || diff < -0.5 {
		t.Errorf("Error: Expected distance %v, got %v", trimmed.Track.Distance, written.Track.Distance)
	}
	if !approxDuration(written.Track.ActivityTime, trimmed.Track.ActivityTime) {
		t.Errorf("Error: Expected activity time %v, got %v", trimmed.Track.ActivityTime, written.Track.ActivityTime)
	}
	if !written.StartTime.Equal(trimmed.StartTime) {
		t.Errorf("Error: Expected start time %v, got %v", trimmed.StartTime, written.StartTime)
	}
}

// Tests WriteGPX() to verify an invalid path returns an error
func TestWriteGPXInvalidPath(t *testing.T) {
	g := makeTrack(500, nil)

	err := g.WriteGPX(filepath.Join(t.TempDir(), "missing_dir", "out.gpx"))
	if err == nil {
		t.Error("Error: Expected Error")
	}
}
