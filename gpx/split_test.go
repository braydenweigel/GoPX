package gpx

import (
	"math"
	"testing"
	"time"
)

// makeTrack builds a GPX with one point every 100m, 30s apart, gaining 1m of elevation per point.
// hr is the heart rate given to every point (nil for no heart rate data).
// The calculated fields are set directly so tests don't depend on ParseFile() or haversine().
func makeTrack(meters float64, hr *int) GPX {
	var points []TrackPoint
	for d := 0.0; d <= meters; d += 100 {
		i := d / 100
		points = append(points, TrackPoint{
			Lat:       d / 100000, //latitude increases with distance so coordinates can be checked
			Lon:       0,
			HeartRate: hr,
			Distance:  d,
			Duration:  time.Duration(i*30) * time.Second,
			ElevGain:  i,
		})
	}
	return gpxFromPoints(points)
}

// gpxFromPoints wraps track points in a GPX, setting the track totals from the last point.
func gpxFromPoints(points []TrackPoint) GPX {
	var g GPX
	g.Track.Segment.TrackPoints = points
	last := points[len(points)-1]
	g.Track.Distance = last.Distance
	g.Track.ActivityTime = last.Duration
	g.Track.ElevationGain = last.ElevGain
	return g
}

func intPtr(i int) *int { return &i }

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func approxDuration(a, b time.Duration) bool {
	diff := a - b
	return diff > -time.Millisecond && diff < time.Millisecond
}

// Tests CalcSplitsDistance() to verify invalid inputs return errors
func TestCalcSplitsDistanceInvalidInputs(t *testing.T) {
	g := makeTrack(4500, nil) //4.5km activity

	tests := []struct {
		name      string
		units     Unit
		start     float64
		intervals int
		length    float64
		partial   bool
	}{
		{"unsupported unit", Unit("yd"), 0, 1, 1, false},
		{"empty unit", Unit(""), 0, 1, 1, false},
		{"zero length", Kilometers, 0, 1, 0, false},
		{"negative length", Kilometers, 0, 1, -1, false},
		{"NaN length", Kilometers, 0, 1, math.NaN(), false},
		{"infinite length", Kilometers, 0, 1, math.Inf(1), false},
		{"negative intervals", Kilometers, 0, -1, 1, false},
		{"negative start", Kilometers, -1, 1, 1, false},
		{"NaN start", Kilometers, math.NaN(), 1, 1, false},
		{"start at end of activity", Kilometers, 4.5, 0, 1, true},
		{"start past end of activity", Kilometers, 5, 0, 1, true},
		{"too many intervals", Kilometers, 0, 5, 1, false},
		{"too many intervals with partial", Kilometers, 0, 5, 1, true}, //partial is ignored when intervals > 0
		{"intervals don't fit after start", Kilometers, 1, 4, 1, false},
		{"no full interval fits", Kilometers, 0, 0, 5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := g.CalcSplitsDistance(tc.units, tc.start, tc.intervals, tc.length, tc.partial)
			if err == nil {
				t.Errorf("Error: Expected error, got %+v", s)
			}
			if s != nil {
				t.Errorf("Error: Expected nil DistanceSplits on error, got %+v", s)
			}
		})
	}
}

// Tests CalcSplitsDistance() to verify intervals that exactly fill the activity are valid
func TestCalcSplitsDistanceExactFit(t *testing.T) {
	g := makeTrack(4000, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 4, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 4 {
		t.Errorf("Error: Expected %v splits, got %v", 4, len(s.Splits))
	}
}

// Tests CalcSplitsDistance() to verify the DistanceSplits fields are set from the inputs
func TestCalcSplitsDistanceFields(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0.5, 2, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	if s.Units != Kilometers {
		t.Errorf("Error: Expected %v, got %v", Kilometers, s.Units)
	}
	if s.Start != 0.5 {
		t.Errorf("Error: Expected %v, got %v", 0.5, s.Start)
	}
	if s.Length != 1 {
		t.Errorf("Error: Expected %v, got %v", 1, s.Length)
	}
	if s.Partial {
		t.Error("Error: Expected Partial to be false")
	}
}

// Tests CalcSplitsDistance() to verify splitting a whole activity without partial drops the leftover distance
func TestCalcSplitsDistanceWholeActivity(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Splits) != 4 {
		t.Fatalf("Error: Expected %v splits, got %v", 4, len(s.Splits))
	}

	for i, split := range s.Splits {
		if !approxEqual(split.Length, 1) {
			t.Errorf("Error: Split %d: Expected length %v, got %v", i, 1, split.Length)
		}
		if !approxDuration(split.Duration, 300*time.Second) { //10 points * 30s
			t.Errorf("Error: Split %d: Expected duration %v, got %v", i, 300*time.Second, split.Duration)
		}
		if !approxEqual(split.ElevGain, 10) { //10 points * 1m
			t.Errorf("Error: Split %d: Expected elevation gain %v, got %v", i, 10, split.ElevGain)
		}
	}
}

// Tests CalcSplitsDistance() to verify splitting a whole activity with partial adds the leftover distance as a final split
func TestCalcSplitsDistancePartial(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	if !s.Partial {
		t.Error("Error: Expected Partial to be true")
	}
	if len(s.Splits) != 5 {
		t.Fatalf("Error: Expected %v splits, got %v", 5, len(s.Splits))
	}

	last := s.Splits[4]
	if !approxEqual(last.Length, 0.5) {
		t.Errorf("Error: Expected length %v, got %v", 0.5, last.Length)
	}
	if !approxDuration(last.Duration, 150*time.Second) {
		t.Errorf("Error: Expected duration %v, got %v", 150*time.Second, last.Duration)
	}
	if !approxEqual(last.EndPoint.Lat, 0.045) { //end of the activity
		t.Errorf("Error: Expected end latitude %v, got %v", 0.045, last.EndPoint.Lat)
	}
}

// Tests CalcSplitsDistance() to verify no partial split is added when the activity divides evenly
func TestCalcSplitsDistancePartialExactFit(t *testing.T) {
	g := makeTrack(4000, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 4 {
		t.Errorf("Error: Expected %v splits, got %v", 4, len(s.Splits))
	}
}

// Tests CalcSplitsDistance() to verify an activity shorter than one interval returns a single partial split
func TestCalcSplitsDistancePartialOnly(t *testing.T) {
	g := makeTrack(600, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 1 {
		t.Fatalf("Error: Expected %v splits, got %v", 1, len(s.Splits))
	}
	if !approxEqual(s.Splits[0].Length, 0.6) {
		t.Errorf("Error: Expected length %v, got %v", 0.6, s.Splits[0].Length)
	}
}

// Tests CalcSplitsDistance() to verify partial is ignored when the number of intervals is specified
func TestCalcSplitsDistancePartialIgnored(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 2, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	if s.Partial {
		t.Error("Error: Expected Partial to be false")
	}
	if len(s.Splits) != 2 {
		t.Errorf("Error: Expected %v splits, got %v", 2, len(s.Splits))
	}
}

// Tests CalcSplitsDistance() to verify splits begin at the start distance and boundaries between points are interpolated
func TestCalcSplitsDistanceStart(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0.25, 2, 1, false) //splits at 250m-1250m and 1250m-2250m
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 2 {
		t.Fatalf("Error: Expected %v splits, got %v", 2, len(s.Splits))
	}

	first, second := s.Splits[0], s.Splits[1]

	if !approxEqual(first.StartPoint.Lat, 0.0025) { //250m, halfway between two points
		t.Errorf("Error: Expected start latitude %v, got %v", 0.0025, first.StartPoint.Lat)
	}
	if !approxEqual(first.EndPoint.Lat, 0.0125) {
		t.Errorf("Error: Expected end latitude %v, got %v", 0.0125, first.EndPoint.Lat)
	}
	if first.EndPoint != second.StartPoint {
		t.Errorf("Error: Expected splits to connect, got %v and %v", first.EndPoint, second.StartPoint)
	}
	if !approxDuration(first.Duration, 300*time.Second) {
		t.Errorf("Error: Expected duration %v, got %v", 300*time.Second, first.Duration)
	}
	if !approxEqual(first.ElevGain, 10) {
		t.Errorf("Error: Expected elevation gain %v, got %v", 10, first.ElevGain)
	}
}

// Tests CalcSplitsDistance() to verify distances are converted to the requested units
func TestCalcSplitsDistanceUnits(t *testing.T) {
	g := makeTrack(4500, nil)

	//miles: 4500m = 2.796mi, so 2 full splits
	s, err := g.CalcSplitsDistance(Miles, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 3 {
		t.Fatalf("Error: Expected %v splits, got %v", 3, len(s.Splits))
	}

	mileDuration := time.Duration(1609.344 / 100 * 30 * float64(time.Second)) //30s per 100m
	if !approxDuration(s.Splits[0].Duration, mileDuration) {
		t.Errorf("Error: Expected duration %v, got %v", mileDuration, s.Splits[0].Duration)
	}
	if !approxEqual(s.Splits[2].Length, 4500/1609.344-2) {
		t.Errorf("Error: Expected length %v, got %v", 4500/1609.344-2, s.Splits[2].Length)
	}

	//feet: 1000ft = 304.8m
	s, err = g.CalcSplitsDistance(Feet, 0, 3, 1000, false)
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(s.Splits[0].ElevGain, 3.048) { //1m per 100m
		t.Errorf("Error: Expected elevation gain %v, got %v", 3.048, s.Splits[0].ElevGain)
	}
}

// Tests CalcSplitsDistance() to verify heart rate is nil when the activity has no heart rate data
func TestCalcSplitsDistanceNoHeartRate(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	for i, split := range s.Splits {
		if split.HeartRate != nil {
			t.Errorf("Error: Split %d: Expected nil heart rate, got %v", i, *split.HeartRate)
		}
	}
}

// Tests CalcSplitsDistance() to verify a constant heart rate averages to itself
func TestCalcSplitsDistanceConstantHeartRate(t *testing.T) {
	g := makeTrack(4500, intPtr(150))

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	for i, split := range s.Splits {
		if split.HeartRate == nil || *split.HeartRate != 150 {
			t.Errorf("Error: Split %d: Expected heart rate %v, got %v", i, 150, split.HeartRate)
		}
	}
}

// Tests CalcSplitsDistance() to verify heart rate is weighted by time, not by number of points
func TestCalcSplitsDistanceHeartRateTimeWeighted(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(100)},
		{Distance: 500, Duration: 100 * time.Second, HeartRate: intPtr(100)},
		{Distance: 1000, Duration: 400 * time.Second, HeartRate: intPtr(200)},
	})

	s, err := g.CalcSplitsDistance(Kilometers, 0, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	//100bpm for 100s, then 100->200bpm (avg 150) for 300s: (100*100 + 150*300) / 400 = 137.5
	//a plain average of the points would be (100+100+200) / 3 = 133
	if s.Splits[0].HeartRate == nil || *s.Splits[0].HeartRate != 138 {
		t.Errorf("Error: Expected heart rate %v, got %v", 138, s.Splits[0].HeartRate)
	}
}

// Tests CalcSplitsDistance() to verify splits shorter than the gap between points still get a heart rate
func TestCalcSplitsDistanceHeartRateShortSplit(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(100)},
		{Distance: 1000, Duration: 300 * time.Second, HeartRate: intPtr(200)},
	})

	s, err := g.CalcSplitsDistance(Kilometers, 0, 4, 0.25, false)
	if err != nil {
		t.Fatal(err)
	}

	//heart rate rises linearly from 100 to 200, so each quarter averages its midpoint
	expected := []int{113, 138, 163, 188} //112.5, 137.5, 162.5, 187.5 rounded
	for i, split := range s.Splits {
		if split.HeartRate == nil || *split.HeartRate != expected[i] {
			t.Errorf("Error: Split %d: Expected heart rate %v, got %v", i, expected[i], split.HeartRate)
		}
	}
}

// Tests CalcSplitsDistance() to verify that if only one of two points has heart rate data, its value is used
func TestCalcSplitsDistanceHeartRateMissingPoint(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(120)},
		{Distance: 1000, Duration: 300 * time.Second, HeartRate: nil},
	})

	s, err := g.CalcSplitsDistance(Kilometers, 0, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Splits[0].HeartRate == nil || *s.Splits[0].HeartRate != 120 {
		t.Errorf("Error: Expected heart rate %v, got %v", 120, s.Splits[0].HeartRate)
	}
}

// Tests CalcSplitsDistance() to verify time spent stopped is included in the split's duration and heart rate
func TestCalcSplitsDistanceStopped(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(150)},
		{Distance: 500, Duration: 100 * time.Second, HeartRate: intPtr(150)},
		{Distance: 500, Duration: 400 * time.Second, HeartRate: intPtr(90)}, //stopped for 300s, heart rate drops
		{Distance: 1000, Duration: 500 * time.Second, HeartRate: intPtr(90)},
	})

	s, err := g.CalcSplitsDistance(Kilometers, 0, 1, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	if !approxDuration(s.Splits[0].Duration, 500*time.Second) {
		t.Errorf("Error: Expected duration %v, got %v", 500*time.Second, s.Splits[0].Duration)
	}

	//150 for 100s, 150->90 (avg 120) for 300s, 90 for 100s: (15000 + 36000 + 9000) / 500 = 120
	if s.Splits[0].HeartRate == nil || *s.Splits[0].HeartRate != 120 {
		t.Errorf("Error: Expected heart rate %v, got %v", 120, s.Splits[0].HeartRate)
	}
}

// Tests CalcSplitsDistance() on a real activity to verify the splits add up to the whole activity
func TestCalcSplitsDistanceFile(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx") //8.05km
	if err != nil {
		t.Fatal(err)
	}

	s, err := g.CalcSplitsDistance(Kilometers, 0, 0, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Splits) != 9 { //8 full splits + 1 partial
		t.Fatalf("Error: Expected %v splits, got %v", 9, len(s.Splits))
	}

	var totalLength, totalElevGain float64
	var totalDuration time.Duration
	for _, split := range s.Splits {
		totalLength += split.Length
		totalDuration += split.Duration
		totalElevGain += split.ElevGain
	}

	if !approxEqual(totalLength*1000, g.Track.Distance) {
		t.Errorf("Error: Expected total length %v, got %v", g.Track.Distance, totalLength*1000)
	}
	if !approxDuration(totalDuration, g.Track.ActivityTime) {
		t.Errorf("Error: Expected total duration %v, got %v", g.Track.ActivityTime, totalDuration)
	}
	if !approxEqual(totalElevGain, g.Track.ElevationGain) {
		t.Errorf("Error: Expected total elevation gain %v, got %v", g.Track.ElevationGain, totalElevGain)
	}
}

// Tests CalcSplitsTime() to verify invalid inputs return errors
func TestCalcSplitsTimeInvalidInputs(t *testing.T) {
	g := makeTrack(4500, nil) //1350s (22.5min) activity

	tests := []struct {
		name      string
		start     float64
		intervals int
		length    float64
		partial   bool
	}{
		{"zero length", 0, 1, 0, false},
		{"NaN length", 0, 1, math.NaN(), false},
		{"infinite length", 0, 1, math.Inf(1), false},
		{"NaN start", math.NaN(), 1, 60, false},
		{"negative length", 0, 1, -60, false},
		{"negative intervals", 0, -1, 60, false},
		{"negative start", -1, 1, 60, false},
		{"start at end of activity", 1350, 0, 60, true},
		{"start past end of activity", 3600, 0, 60, true},
		{"too many intervals", 0, 3, 600, false},
		{"too many intervals with partial", 0, 3, 600, true}, //partial is ignored when intervals > 0
		{"intervals don't fit after start", 300, 2, 600, false},
		{"no full interval fits", 0, 0, 3600, false},
		{"huge interval count", 0, math.MaxInt, 3600, false}, //must not overflow into a passing check
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := g.CalcSplitsTime(tc.start, tc.intervals, tc.length, tc.partial)
			if err == nil {
				t.Errorf("Error: Expected error, got %+v", s)
			}
			if s != nil {
				t.Errorf("Error: Expected nil TimeSplits on error, got %+v", s)
			}
		})
	}
}

// Tests CalcSplitsTime() to verify splitting a whole activity with and without partial
func TestCalcSplitsTimeWholeActivity(t *testing.T) {
	g := makeTrack(4500, nil) //1350s activity, 100m every 30s

	s, err := g.CalcSplitsTime(0, 0, 300, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Splits) != 4 {
		t.Fatalf("Error: Expected %v splits, got %v", 4, len(s.Splits))
	}
	for i, split := range s.Splits {
		if !approxDuration(split.Duration, 5*time.Minute) {
			t.Errorf("Error: Split %d: Expected duration %v, got %v", i, 5*time.Minute, split.Duration)
		}
		if !approxEqual(split.Length, 1000) { //300s at 100m/30s
			t.Errorf("Error: Split %d: Expected length %v, got %v", i, 1000, split.Length)
		}
		if !approxEqual(split.ElevGain, 10) {
			t.Errorf("Error: Split %d: Expected elevation gain %v, got %v", i, 10, split.ElevGain)
		}
	}

	s, err = g.CalcSplitsTime(0, 0, 300, true)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Partial {
		t.Error("Error: Expected Partial to be true")
	}
	if len(s.Splits) != 5 {
		t.Fatalf("Error: Expected %v splits, got %v", 5, len(s.Splits))
	}
	last := s.Splits[4]
	if !approxDuration(last.Duration, 150*time.Second) {
		t.Errorf("Error: Expected duration %v, got %v", 150*time.Second, last.Duration)
	}
	if !approxEqual(last.Length, 500) {
		t.Errorf("Error: Expected length %v, got %v", 500, last.Length)
	}
}

// Tests CalcSplitsTime() to verify partial is ignored when the number of intervals is specified
func TestCalcSplitsTimePartialIgnored(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsTime(0, 2, 300, true)
	if err != nil {
		t.Fatal(err)
	}
	if s.Partial {
		t.Error("Error: Expected Partial to be false")
	}
	if len(s.Splits) != 2 {
		t.Errorf("Error: Expected %v splits, got %v", 2, len(s.Splits))
	}
}

// Tests CalcSplitsTime() to verify splits begin at the start time and boundaries between points are interpolated
func TestCalcSplitsTimeStart(t *testing.T) {
	g := makeTrack(4500, nil)

	s, err := g.CalcSplitsTime(45, 1, 60, false) //45s-105s = 150m-350m
	if err != nil {
		t.Fatal(err)
	}

	split := s.Splits[0]
	if !approxEqual(split.StartPoint.Lat, 0.0015) {
		t.Errorf("Error: Expected start latitude %v, got %v", 0.0015, split.StartPoint.Lat)
	}
	if !approxEqual(split.EndPoint.Lat, 0.0035) {
		t.Errorf("Error: Expected end latitude %v, got %v", 0.0035, split.EndPoint.Lat)
	}
	if !approxEqual(split.Length, 200) {
		t.Errorf("Error: Expected length %v, got %v", 200, split.Length)
	}
}

// Tests CalcSplitsTime() to verify heart rate is weighted by time
func TestCalcSplitsTimeHeartRate(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0, HeartRate: intPtr(100)},
		{Distance: 1000, Duration: 400 * time.Second, HeartRate: intPtr(200)},
	})

	s, err := g.CalcSplitsTime(0, 2, 200, false)
	if err != nil {
		t.Fatal(err)
	}

	//heart rate rises linearly from 100 to 200, so each half averages its midpoint
	expected := []int{125, 175}
	for i, split := range s.Splits {
		if split.HeartRate == nil || *split.HeartRate != expected[i] {
			t.Errorf("Error: Split %d: Expected heart rate %v, got %v", i, expected[i], split.HeartRate)
		}
	}
}

// Tests CalcSplitsTime() to verify time spent stopped counts towards the split it happens in
func TestCalcSplitsTimeStopped(t *testing.T) {
	g := gpxFromPoints([]TrackPoint{
		{Distance: 0, Duration: 0},
		{Distance: 500, Duration: 100 * time.Second},
		{Distance: 500, Duration: 400 * time.Second}, //stopped for 300s
		{Distance: 1000, Duration: 500 * time.Second},
	})

	s, err := g.CalcSplitsTime(0, 0, 250, false)
	if err != nil {
		t.Fatal(err)
	}

	//0-250s: 500m (all of the moving) then stopped. 250-500s: stopped until 400s, then 500m
	expected := []float64{500, 500}
	for i, split := range s.Splits {
		if !approxEqual(split.Length, expected[i]) {
			t.Errorf("Error: Split %d: Expected length %v, got %v", i, expected[i], split.Length)
		}
	}
}

// Tests CalcSplitsTime() on a real activity to verify the splits add up to the whole activity
func TestCalcSplitsTimeFile(t *testing.T) {
	g, err := ParseFile("testdata/test1.gpx") //1644s
	if err != nil {
		t.Fatal(err)
	}

	s, err := g.CalcSplitsTime(0, 0, 300, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Splits) != 6 { //5 full splits + 1 partial (144s)
		t.Fatalf("Error: Expected %v splits, got %v", 6, len(s.Splits))
	}

	var totalLength, totalElevGain float64
	var totalDuration time.Duration
	for _, split := range s.Splits {
		totalLength += split.Length
		totalDuration += split.Duration
		totalElevGain += split.ElevGain
	}

	if !approxEqual(totalLength, g.Track.Distance) {
		t.Errorf("Error: Expected total length %v, got %v", g.Track.Distance, totalLength)
	}
	if !approxDuration(totalDuration, g.Track.ActivityTime) {
		t.Errorf("Error: Expected total duration %v, got %v", g.Track.ActivityTime, totalDuration)
	}
	if !approxEqual(totalElevGain, g.Track.ElevationGain) {
		t.Errorf("Error: Expected total elevation gain %v, got %v", g.Track.ElevationGain, totalElevGain)
	}
}
