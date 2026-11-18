package gpx

import (
	"math"
	"testing"
	"time"
)

// Tests TrimDistance() to verify invalid inputs return errors
func TestTrimDistanceInvalidInputs(t *testing.T) {
	g := makeTrack(4500, nil) //4.5km activity

	tests := []struct {
		name      string
		units     Unit
		fromStart float64
		fromEnd   float64
	}{
		{"unsupported unit", Unit("yd"), 1, 0},
		{"negative start", Kilometers, -1, 0},
		{"negative end", Kilometers, 0, -1},
		{"NaN start", Kilometers, math.NaN(), 0},
		{"infinite end", Kilometers, 0, math.Inf(1)},
		{"trim whole activity from start", Kilometers, 4.5, 0},
		{"trim whole activity from end", Kilometers, 0, 4.5},
		{"trim more than activity", Kilometers, 3, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trimmed, err := g.TrimDistance(tc.units, tc.fromStart, tc.fromEnd)
			if err == nil {
				t.Errorf("Error: Expected error, got %+v", trimmed)
			}
			if trimmed != nil {
				t.Errorf("Error: Expected nil GPX on error, got %+v", trimmed)
			}
		})
	}
}

// Tests TrimDistance() to verify trimming the start interpolates a new first point and resets the totals
func TestTrimDistanceStart(t *testing.T) {
	g := makeTrack(4500, nil) //100m every 30s, 1m elevation gain per point

	trimmed, err := g.TrimDistance(Kilometers, 0.25, 0)
	if err != nil {
		t.Fatal(err)
	}

	points := trimmed.Track.Segment.TrackPoints
	first, last := points[0], points[len(points)-1]

	if !approxEqual(first.Lat, 0.0025) { //250m, halfway between two points
		t.Errorf("Error: Expected first latitude %v, got %v", 0.0025, first.Lat)
	}
	if first.Distance != 0 || first.Duration != 0 || first.ElevGain != 0 {
		t.Errorf("Error: Expected first point totals to be 0, got %v, %v, %v", first.Distance, first.Duration, first.ElevGain)
	}
	if !approxEqual(last.Lat, 0.045) { //end of the activity is untouched
		t.Errorf("Error: Expected last latitude %v, got %v", 0.045, last.Lat)
	}
	if !approxEqual(trimmed.Track.Distance, 4250) {
		t.Errorf("Error: Expected distance %v, got %v", 4250, trimmed.Track.Distance)
	}
	if !approxDuration(trimmed.Track.ActivityTime, 1275*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 1275*time.Second, trimmed.Track.ActivityTime)
	}
	if !approxEqual(trimmed.Track.ElevationGain, 42.5) {
		t.Errorf("Error: Expected elevation gain %v, got %v", 42.5, trimmed.Track.ElevationGain)
	}
	if len(points) != 44 { //interpolated point at 250m + original points 300m-4500m
		t.Errorf("Error: Expected %v points, got %v", 44, len(points))
	}
}

// Tests TrimDistance() to verify trimming the end interpolates a new last point
func TestTrimDistanceEnd(t *testing.T) {
	g := makeTrack(4500, nil)

	trimmed, err := g.TrimDistance(Kilometers, 0, 0.25)
	if err != nil {
		t.Fatal(err)
	}

	points := trimmed.Track.Segment.TrackPoints
	if !approxEqual(points[len(points)-1].Lat, 0.0425) { //4250m
		t.Errorf("Error: Expected last latitude %v, got %v", 0.0425, points[len(points)-1].Lat)
	}
	if points[0].Lat != 0 {
		t.Errorf("Error: Expected first latitude %v, got %v", 0, points[0].Lat)
	}
	if !approxEqual(trimmed.Track.Distance, 4250) {
		t.Errorf("Error: Expected distance %v, got %v", 4250, trimmed.Track.Distance)
	}
	if !approxDuration(trimmed.Track.ActivityTime, 1275*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 1275*time.Second, trimmed.Track.ActivityTime)
	}
}

// Tests TrimDistance() to verify trimming both ends, including cuts that land exactly on a point
func TestTrimDistanceBoth(t *testing.T) {
	g := makeTrack(4500, nil)

	trimmed, err := g.TrimDistance(Kilometers, 1, 0.5) //keeps 1000m-4000m
	if err != nil {
		t.Fatal(err)
	}

	points := trimmed.Track.Segment.TrackPoints
	if len(points) != 31 { //1000m, 1100m, ..., 4000m. no duplicate points at the cuts
		t.Errorf("Error: Expected %v points, got %v", 31, len(points))
	}
	if !approxEqual(trimmed.Track.Distance, 3000) {
		t.Errorf("Error: Expected distance %v, got %v", 3000, trimmed.Track.Distance)
	}
	if !approxEqual(trimmed.Track.ElevationGain, 30) {
		t.Errorf("Error: Expected elevation gain %v, got %v", 30, trimmed.Track.ElevationGain)
	}
}

// Tests TrimDistance() to verify distances are converted from the requested units
func TestTrimDistanceUnits(t *testing.T) {
	g := makeTrack(4500, nil)

	trimmed, err := g.TrimDistance(Miles, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(trimmed.Track.Distance, 4500-1609.344) {
		t.Errorf("Error: Expected distance %v, got %v", 4500-1609.344, trimmed.Track.Distance)
	}

	trimmed, err = g.TrimDistance(Feet, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(trimmed.Track.Distance, 4500-304.8) {
		t.Errorf("Error: Expected distance %v, got %v", 4500-304.8, trimmed.Track.Distance)
	}
}

// Tests TrimDistance() to verify the original GPX is not modified
func TestTrimDistanceOriginalUnchanged(t *testing.T) {
	g := makeTrack(4500, nil)

	trimmed, err := g.TrimDistance(Kilometers, 0, 0) //trims nothing, so the points could easily be shared by mistake
	if err != nil {
		t.Fatal(err)
	}

	trimmed.Track.Segment.TrackPoints[1].Distance = -1

	if g.Track.Segment.TrackPoints[1].Distance != 100 {
		t.Errorf("Error: Expected original point to be unchanged, got distance %v", g.Track.Segment.TrackPoints[1].Distance)
	}
	if len(g.Track.Segment.TrackPoints) != 46 || g.Track.Distance != 4500 {
		t.Errorf("Error: Expected original track to be unchanged, got %v points and %vm", len(g.Track.Segment.TrackPoints), g.Track.Distance)
	}
}

// Tests TrimDistance() to verify the start time and point times move with the trim
func TestTrimDistanceTimes(t *testing.T) {
	start := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, Time: start},
		{Distance: 1000, Duration: 300 * time.Second, Time: start.Add(300 * time.Second)},
	})
	g.StartTime = start

	trimmed, err := g.TrimDistance(Kilometers, 0.5, 0)
	if err != nil {
		t.Fatal(err)
	}

	expected := start.Add(150 * time.Second)
	if !trimmed.StartTime.Equal(expected) {
		t.Errorf("Error: Expected start time %v, got %v", expected, trimmed.StartTime)
	}
	if !trimmed.Track.Segment.TrackPoints[0].Time.Equal(expected) {
		t.Errorf("Error: Expected first point time %v, got %v", expected, trimmed.Track.Segment.TrackPoints[0].Time)
	}
}

// Tests TrimDistance() to verify the cut point's heart rate is interpolated
func TestTrimDistanceHeartRate(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(100)},
		{Distance: 1000, Duration: 300 * time.Second, HeartRate: intPtr(200)},
	})

	trimmed, err := g.TrimDistance(Kilometers, 0.25, 0)
	if err != nil {
		t.Fatal(err)
	}

	hr := trimmed.Track.Segment.TrackPoints[0].HeartRate
	if hr == nil || *hr != 125 {
		t.Errorf("Error: Expected heart rate %v, got %v", 125, hr)
	}
}

// Tests TrimDistance() to verify time stopped exactly at a cut is trimmed off
func TestTrimDistanceStopped(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0},
		{Distance: 500, Duration: 100 * time.Second},
		{Distance: 500, Duration: 400 * time.Second}, //stopped for 300s
		{Distance: 1000, Duration: 500 * time.Second},
	})

	//start cut at the stop: starts when movement resumes
	trimmed, err := g.TrimDistance(Kilometers, 0.5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !approxDuration(trimmed.Track.ActivityTime, 100*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 100*time.Second, trimmed.Track.ActivityTime)
	}

	//end cut at the stop: ends on arrival
	trimmed, err = g.TrimDistance(Kilometers, 0, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if !approxDuration(trimmed.Track.ActivityTime, 100*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 100*time.Second, trimmed.Track.ActivityTime)
	}
}

// Tests TrimDistance() on a real activity
func TestTrimDistanceFile(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx")
	if err != nil {
		t.Fatal(err)
	}

	trimmed, err := g.TrimDistance(Kilometers, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !approxEqual(trimmed.Track.Distance, g.Track.Distance-2000) {
		t.Errorf("Error: Expected distance %v, got %v", g.Track.Distance-2000, trimmed.Track.Distance)
	}
	if trimmed.Track.ActivityTime >= g.Track.ActivityTime {
		t.Errorf("Error: Expected activity time less than %v, got %v", g.Track.ActivityTime, trimmed.Track.ActivityTime)
	}
	if trimmed.Track.Name != g.Track.Name || trimmed.Track.Type != g.Track.Type {
		t.Error("Error: Expected track name and type to be copied")
	}
}

// Tests TrimTime() to verify invalid inputs return errors
func TestTrimTimeInvalidInputs(t *testing.T) {
	g := makeTrack(4500, nil) //1350s activity

	tests := []struct {
		name      string
		fromStart float64
		fromEnd   float64
	}{
		{"negative start", -1, 0},
		{"negative end", 0, -1},
		{"NaN start", math.NaN(), 0},
		{"infinite end", 0, math.Inf(1)},
		{"trim whole activity from start", 1350, 0},
		{"trim whole activity from end", 0, 1350},
		{"trim more than activity", 1000, 400},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trimmed, err := g.TrimTime(tc.fromStart, tc.fromEnd)
			if err == nil {
				t.Errorf("Error: Expected error, got %+v", trimmed)
			}
			if trimmed != nil {
				t.Errorf("Error: Expected nil GPX on error, got %+v", trimmed)
			}
		})
	}
}

// Tests TrimTime() to verify trimming both ends interpolates new end points and resets the totals
func TestTrimTimeBoth(t *testing.T) {
	g := makeTrack(4500, nil) //100m every 30s, 1m elevation gain per point

	trimmed, err := g.TrimTime(45, 75) //keeps 45s-1275s = 150m-4250m
	if err != nil {
		t.Fatal(err)
	}

	points := trimmed.Track.Segment.TrackPoints
	first, last := points[0], points[len(points)-1]

	if !approxEqual(first.Lat, 0.0015) {
		t.Errorf("Error: Expected first latitude %v, got %v", 0.0015, first.Lat)
	}
	if !approxEqual(last.Lat, 0.0425) {
		t.Errorf("Error: Expected last latitude %v, got %v", 0.0425, last.Lat)
	}
	if first.Distance != 0 || first.Duration != 0 || first.ElevGain != 0 {
		t.Errorf("Error: Expected first point totals to be 0, got %v, %v, %v", first.Distance, first.Duration, first.ElevGain)
	}
	if !approxDuration(trimmed.Track.ActivityTime, 1230*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 1230*time.Second, trimmed.Track.ActivityTime)
	}
	if !approxEqual(trimmed.Track.Distance, 4100) {
		t.Errorf("Error: Expected distance %v, got %v", 4100, trimmed.Track.Distance)
	}
	if !approxEqual(trimmed.Track.ElevationGain, 41) {
		t.Errorf("Error: Expected elevation gain %v, got %v", 41, trimmed.Track.ElevationGain)
	}
}

// Tests TrimTime() to verify trimming only one end leaves the other untouched
func TestTrimTimeOneEnd(t *testing.T) {
	g := makeTrack(4500, nil)

	trimmed, err := g.TrimTime(300, 0)
	if err != nil {
		t.Fatal(err)
	}
	points := trimmed.Track.Segment.TrackPoints
	if len(points) != 36 { //1000m-4500m, cut lands exactly on a point
		t.Errorf("Error: Expected %v points, got %v", 36, len(points))
	}
	if !approxEqual(points[len(points)-1].Lat, 0.045) {
		t.Errorf("Error: Expected last latitude %v, got %v", 0.045, points[len(points)-1].Lat)
	}

	trimmed, err = g.TrimTime(0, 300)
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.Track.Segment.TrackPoints[0].Lat != 0 {
		t.Errorf("Error: Expected first latitude %v, got %v", 0, trimmed.Track.Segment.TrackPoints[0].Lat)
	}
	if !approxEqual(trimmed.Track.Distance, 3500) {
		t.Errorf("Error: Expected distance %v, got %v", 3500, trimmed.Track.Distance)
	}
}

// Tests TrimTime() to verify the start time moves with the trim and the original is unchanged
func TestTrimTimeStartTime(t *testing.T) {
	start := time.Date(2026, 3, 15, 16, 0, 0, 0, time.UTC)
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, Time: start, HeartRate: intPtr(100)},
		{Distance: 1000, Duration: 300 * time.Second, Time: start.Add(300 * time.Second), HeartRate: intPtr(200)},
	})
	g.StartTime = start

	trimmed, err := g.TrimTime(60, 0)
	if err != nil {
		t.Fatal(err)
	}

	expected := start.Add(60 * time.Second)
	if !trimmed.StartTime.Equal(expected) {
		t.Errorf("Error: Expected start time %v, got %v", expected, trimmed.StartTime)
	}
	if hr := trimmed.Track.Segment.TrackPoints[0].HeartRate; hr == nil || *hr != 120 {
		t.Errorf("Error: Expected heart rate %v, got %v", 120, hr)
	}
	if !g.StartTime.Equal(start) || g.Track.Segment.TrackPoints[0].Distance != 0 || len(g.Track.Segment.TrackPoints) != 2 {
		t.Error("Error: Expected original GPX to be unchanged")
	}
}

// Tests TrimTime() on a real activity
func TestTrimTimeFile(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx") //1644s
	if err != nil {
		t.Fatal(err)
	}

	trimmed, err := g.TrimTime(120, 60)
	if err != nil {
		t.Fatal(err)
	}

	if !approxDuration(trimmed.Track.ActivityTime, 1464*time.Second) {
		t.Errorf("Error: Expected activity time %v, got %v", 1464*time.Second, trimmed.Track.ActivityTime)
	}
	if trimmed.Track.Distance >= g.Track.Distance {
		t.Errorf("Error: Expected distance less than %v, got %v", g.Track.Distance, trimmed.Track.Distance)
	}
}

// Tests TrimDistance() and TrimTime() to verify net elevation change is recalculated from the new end points
func TestTrimNetElev(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, Elevation: 100},
		{Distance: 1000, Duration: 300 * time.Second, Elevation: 120},
		{Distance: 2000, Duration: 600 * time.Second, Elevation: 90},
	})

	trimmed, err := g.TrimDistance(Kilometers, 0.5, 0) //starts at 110m elevation
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(trimmed.Track.NetElev, -20) {
		t.Errorf("Error: Expected net elevation %v, got %v", -20, trimmed.Track.NetElev)
	}

	trimmed, err = g.TrimTime(0, 300) //ends at 120m elevation
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(trimmed.Track.NetElev, 20) {
		t.Errorf("Error: Expected net elevation %v, got %v", 20, trimmed.Track.NetElev)
	}
}
