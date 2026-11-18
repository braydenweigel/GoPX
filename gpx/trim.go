package gpx

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// TrimDistance returns a copy of the activity with fromStart removed from the beginning
// and fromEnd removed from the end, both in units. The original GPX is not modified.
// Use 0 to leave an end untouched.
func (g GPX) TrimDistance(units Unit, fromStart float64, fromEnd float64) (*GPX, error) {
	//check for errors

	//units must be mi, km, or ft. also gets the conversion factor from meters
	metersPerUnit, err := units.meters()
	if err != nil {
		return nil, err
	}

	//amounts to trim must be finite and not negative
	if math.IsNaN(fromStart) || math.IsInf(fromStart, 0) || fromStart < 0 {
		return nil, fmt.Errorf("distance to trim from start must be 0 or more, got %v", fromStart)
	}
	if math.IsNaN(fromEnd) || math.IsInf(fromEnd, 0) || fromEnd < 0 {
		return nil, fmt.Errorf("distance to trim from end must be 0 or more, got %v", fromEnd)
	}

	total := g.Track.Distance / metersPerUnit // activity distance in the requested units

	//something has to be left after trimming
	if fromStart+fromEnd >= total {
		return nil, fmt.Errorf("trimming %v%s from the start and %v%s from the end leaves nothing of a %.2f%s activity",
			fromStart, units, fromEnd, units, total, units)
	}

	//cut points in meters from the start of the activity
	cutStart := fromStart * metersPerUnit
	cutEnd := g.Track.Distance - fromEnd*metersPerUnit

	return g.trim(byDistance, cutStart, cutEnd)
}

// TrimTime returns a copy of the activity with fromStart seconds removed from the beginning
// and fromEnd seconds removed from the end. The original GPX is not modified.
// Use 0 to leave an end untouched.
func (g GPX) TrimTime(fromStart float64, fromEnd float64) (*GPX, error) {
	//check for errors

	//amounts to trim must be finite and not negative
	if math.IsNaN(fromStart) || math.IsInf(fromStart, 0) || fromStart < 0 {
		return nil, fmt.Errorf("time to trim from start must be 0 or more seconds, got %v", fromStart)
	}
	if math.IsNaN(fromEnd) || math.IsInf(fromEnd, 0) || fromEnd < 0 {
		return nil, fmt.Errorf("time to trim from end must be 0 or more seconds, got %v", fromEnd)
	}

	total := g.Track.ActivityTime.Seconds() // activity time in seconds

	//something has to be left after trimming
	if fromStart+fromEnd >= total {
		return nil, fmt.Errorf("trimming %vs from the start and %vs from the end leaves nothing of a %vs activity",
			fromStart, fromEnd, total)
	}

	//cut points in seconds from the start of the activity
	cutStart := fromStart
	cutEnd := total - fromEnd

	return g.trim(byTime, cutStart, cutEnd)
}

// trim returns a copy of the activity keeping only the part between v0 and v1, measured along key
// (byDistance for meters, byTime for seconds). The callers check that 0 <= v0 < v1 <= the activity total.
func (g GPX) trim(key func(TrackPoint) float64, v0, v1 float64) (*GPX, error) {
	points := trimPoints(g.Track.Segment.TrackPoints, key, v0, v1)
	if len(points) < 2 {
		return nil, errors.New("trimmed activity must have at least 2 track points")
	}

	//shift the running totals so they count from the new first point
	base := points[0]
	for i := range points {
		points[i].Distance -= base.Distance
		points[i].Duration -= base.Duration
		points[i].ElevGain -= base.ElevGain
	}

	trimmed := g //copies the fields. the points slice is replaced below, so the original's points aren't shared
	trimmed.Track.Segment.TrackPoints = points

	last := points[len(points)-1]
	trimmed.Track.Distance = last.Distance
	trimmed.Track.ActivityTime = last.Duration
	trimmed.Track.ElevationGain = last.ElevGain
	trimmed.Track.NetElev = last.Elevation - points[0].Elevation

	//move the start time forward by the time trimmed off. left alone if the file had no start time
	if !g.StartTime.IsZero() {
		trimmed.StartTime = g.StartTime.Add(base.Duration)
	}

	return &trimmed, nil
}

// trimPoints returns a new slice with the points between v0 and v1, measured along key.
// A point is interpolated at each cut so the trimmed track starts and ends exactly at v0 and v1.
// If v0 or v1 is at or past the end of the track, that end is kept as-is (including any time stopped there).
// When a cut lands where the activity was stopped, the stopped time is trimmed off: the start cut uses
// the moment movement resumes, and the end cut uses the moment the activity arrived there.
func trimPoints(points []TrackPoint, key func(TrackPoint) float64, v0, v1 float64) []TrackPoint {
	if len(points) == 0 {
		return nil
	}

	result := make([]TrackPoint, 0, len(points)+2)

	//start: the first point, or a point interpolated at v0
	next := 1 //index of the first original point after the start
	if v0 <= key(points[0]) {
		result = append(result, points[0])
	} else {
		//first point past v0. points[next-1] is the last point at or before it
		for next < len(points) && key(points[next]) <= v0 {
			next++
		}
		if next == len(points) {
			return nil //v0 is at or past the end of the track
		}
		a, b := points[next-1], points[next]
		result = append(result, interpolatePoint(a, b, (v0-key(a))/(key(b)-key(a))))
	}

	//end: every remaining point, or the points before v1 plus a point interpolated at v1
	if v1 >= key(points[len(points)-1]) {
		return append(result, points[next:]...)
	}

	end := next //first point at or past v1
	for key(points[end]) < v1 {
		end++
	}
	result = append(result, points[next:end]...)

	if key(points[end]) == v1 {
		return append(result, points[end])
	}
	a, b := points[end-1], points[end]
	return append(result, interpolatePoint(a, b, (v1-key(a))/(key(b)-key(a))))
}

// interpolatePoint returns a point t of the way from a to b (0 = a, 1 = b).
func interpolatePoint(a, b TrackPoint, t float64) TrackPoint {
	lerp := func(x, y float64) float64 { return x + t*(y-x) }

	p := TrackPoint{
		Lat:       lerp(a.Lat, b.Lat),
		Lon:       lerp(a.Lon, b.Lon),
		Elevation: lerp(a.Elevation, b.Elevation),
		Time:      a.Time.Add(time.Duration(t * float64(b.Time.Sub(a.Time)))),
		Distance:  lerp(a.Distance, b.Distance),
		Duration:  a.Duration + time.Duration(t*float64(b.Duration-a.Duration)),
		ElevGain:  lerp(a.ElevGain, b.ElevGain),
	}

	//heart rate: interpolated if both points have it, otherwise whichever one does
	switch {
	case a.HeartRate != nil && b.HeartRate != nil:
		hr := int(math.Round(lerp(float64(*a.HeartRate), float64(*b.HeartRate))))
		p.HeartRate = &hr
	case a.HeartRate != nil:
		hr := *a.HeartRate
		p.HeartRate = &hr
	case b.HeartRate != nil:
		hr := *b.HeartRate
		p.HeartRate = &hr
	}

	return p
}
