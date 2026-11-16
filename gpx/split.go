package gpx

import (
	"fmt"
	"math"
	"time"
)

func (g GPX) CalcSplitsDistance(units Unit, start float64, intervals int, length float64, partial bool) (*DistanceSplits, error) {
	//check for errors

	//units must be mi, km, or ft. also gets the conversion factor from meters
	metersPerUnit, err := units.meters()
	if err != nil {
		return nil, err
	}

	//split length must be a positive, finite number
	if math.IsNaN(length) || math.IsInf(length, 0) || length <= 0 {
		return nil, fmt.Errorf("split length must be greater than 0, got %v", length)
	}

	//intervals can't be negative. 0 means "as many full intervals as fit"
	if intervals < 0 {
		return nil, fmt.Errorf("number of intervals must be 0 (as many as fit) or more, got %d", intervals)
	}

	//partial only applies when splitting the whole activity (intervals == 0). ignore it otherwise
	if intervals > 0 {
		partial = false
	}

	total := g.Track.Distance / metersPerUnit // activity distance in the requested units

	//start must be a finite number that falls within the activity (0 <= start < total)
	if math.IsNaN(start) || math.IsInf(start, 0) || start < 0 || start >= total {
		return nil, fmt.Errorf("start must be at least 0 and less than the activity distance (%.2f%s), got %v", total, units, start)
	}

	//intervals == 0: use as many full intervals as fit between start and the end of the activity
	if intervals == 0 {
		intervals = int((total - start) / length)

		//there must be at least one split to return: a full interval, or a partial one if allowed
		if intervals == 0 && !partial {
			return nil, fmt.Errorf("no full %v%s interval fits after %v%s in a %.2f%s activity",
				length, units, start, units, total, units)
		}
	}

	//all intervals must fit completely between start and the end of the activity
	//(5x1km intervals need at least 5km after start)
	if start+float64(intervals)*length > total {
		return nil, fmt.Errorf("%d intervals of %v%s starting at %v%s do not fit in a %.2f%s activity",
			intervals, length, units, start, units, total, units)
	}

	//initialize DistanceSplits object
	var s DistanceSplits

	s.Units = units
	s.Length = length
	s.Start = start
	s.Partial = partial

	//create s.Splits
	points := g.Track.Segment.TrackPoints

	//split boundaries in meters (the units TrackPoint.Distance uses)
	boundaries := make([]float64, 0, intervals+2)
	for i := 0; i <= intervals; i++ {
		boundaries = append(boundaries, (start+float64(i)*length)*metersPerUnit)
	}

	//partial split: whatever is left between the last full split and the end of the activity
	if partial && g.Track.Distance > boundaries[len(boundaries)-1] {
		boundaries = append(boundaries, g.Track.Distance)
	}

	s.Splits = make([]Split, 0, len(boundaries)-1)

	seg := 0   //index of the track segment used for interpolation. only moves forward
	hrSeg := 0 //index of the first track segment to check for heart rate. only moves forward

	startPos := positionAt(points, boundaries[0], &seg)

	for i := 1; i < len(boundaries); i++ {
		endPos := positionAt(points, boundaries[i], &seg)
		last := i == len(boundaries)-1

		heartRate := avgHeartRate(points, boundaries[i-1], boundaries[i], last, &hrSeg)

		s.Splits = append(s.Splits, Split{
			Length:     (boundaries[i] - boundaries[i-1]) / metersPerUnit,
			Duration:   endPos.duration - startPos.duration,
			StartPoint: startPos.coord,
			EndPoint:   endPos.coord,
			ElevGain:   endPos.elevGain - startPos.elevGain,
			HeartRate:  heartRate,
		})

		startPos = endPos
	}

	return &s, nil
}

// avgHeartRate returns the time-weighted average heart rate between distances d0 and d1 (meters),
// or nil if there is no heart rate data in that range.
// Between two track points, heart rate is assumed to change linearly over time. If only one of the
// two points has a heart rate, that value is used for the whole segment.
// last includes time spent stopped exactly at d1 (only the final split should do this).
// seg is the index of the segment to start searching from. it's updated so repeated calls
// with increasing distances don't rescan the track.
func avgHeartRate(points []TrackPoint, d0, d1 float64, last bool, seg *int) *int {
	//skip segments that end before the split starts
	for *seg < len(points)-1 && points[*seg+1].Distance < d0 {
		*seg++
	}

	var weightedSum float64 //sum of heart rate * seconds
	var totalTime float64   //seconds with heart rate data

	for j := *seg; j < len(points)-1 && points[j].Distance <= d1; j++ {
		a, b := points[j], points[j+1]
		if a.HeartRate == nil && b.HeartRate == nil {
			continue
		}

		//portion of the segment (0 = a, 1 = b) that falls inside the split
		var t0, t1 float64
		if span := b.Distance - a.Distance; span > 0 {
			t0 = math.Max(0, (d0-a.Distance)/span)
			t1 = math.Min(1, (d1-a.Distance)/span)
			if t1 <= t0 {
				continue
			}
		} else {
			//stopped: no distance covered. count the time in the split that contains this distance
			if a.Distance < d0 || a.Distance > d1 || (a.Distance == d1 && !last) {
				continue
			}
			t0, t1 = 0, 1
		}

		dt := (t1 - t0) * (b.Duration - a.Duration).Seconds()
		if dt <= 0 {
			continue
		}

		//average heart rate over the portion
		var hr float64
		switch {
		case a.HeartRate == nil:
			hr = float64(*b.HeartRate)
		case b.HeartRate == nil:
			hr = float64(*a.HeartRate)
		default:
			hrA, hrB := float64(*a.HeartRate), float64(*b.HeartRate)
			hr = hrA + (t0+t1)/2*(hrB-hrA)
		}

		weightedSum += hr * dt
		totalTime += dt
	}

	if totalTime == 0 {
		return nil
	}

	avg := int(math.Round(weightedSum / totalTime))
	return &avg
}

// position is the interpolated state of the activity at a given distance.
type position struct {
	coord    Coordinate
	duration time.Duration //time since the start of the activity
	elevGain float64       //elevation gain since the start of the activity
}

// positionAt linearly interpolates between the two track points on either side of dist (meters).
// seg is the index of the segment to start searching from. it's updated so repeated calls
// with increasing distances don't rescan the track.
func positionAt(points []TrackPoint, dist float64, seg *int) position {
	if len(points) == 1 {
		p := points[0]
		return position{Coordinate{p.Lat, p.Lon}, p.Duration, p.ElevGain}
	}

	//find the segment points[seg] -> points[seg+1] that contains dist
	for *seg < len(points)-2 && points[*seg+1].Distance < dist {
		*seg++
	}

	a, b := points[*seg], points[*seg+1]

	//fraction of the way from a to b. clamped so distances past either end use the end point
	t := 0.0
	if span := b.Distance - a.Distance; span > 0 {
		t = math.Max(0, math.Min(1, (dist-a.Distance)/span))
	}

	return position{
		coord: Coordinate{
			Lat: a.Lat + t*(b.Lat-a.Lat),
			Lon: a.Lon + t*(b.Lon-a.Lon),
		},
		duration: a.Duration + time.Duration(t*float64(b.Duration-a.Duration)),
		elevGain: a.ElevGain + t*(b.ElevGain-a.ElevGain),
	}
}

// meters returns the number of meters in one of the unit, or an error if the unit is not supported.
func (u Unit) meters() (float64, error) {
	switch u {
	case Miles:
		return 1609.344, nil
	case Kilometers:
		return 1000, nil
	case Feet:
		return 0.3048, nil
	default:
		return 0, fmt.Errorf("unsupported unit %q: must be %q, %q, or %q", u, Miles, Kilometers, Feet)
	}
}
