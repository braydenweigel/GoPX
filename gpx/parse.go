package gpx

import (
	"encoding/xml"
	"errors"
	"math"
	"os"
)

func ParseFile(path string) (*GPX, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var gpx GPX

	err = xml.Unmarshal(data, &gpx)
	if err != nil {
		return nil, err
	}

	//Calculate Activity Statistics
	err = calcStats(&gpx.Track)
	if err != nil {
		return nil, err
	}

	return &gpx, nil
}

// calcStats fills in the calculated fields of each track point (running totals from the start)
// and the track's totals.
func calcStats(track *Track) error {
	points := track.Segment.TrackPoints
	if len(points) == 0 {
		return errors.New("No Data")
	}

	startTime := points[0].Time
	var totalDistance float64 = 0
	var totalElevation float64 = 0

	for i := range points {
		if i > 0 {
			prev := points[i-1]

			segmentDistance := haversine(
				prev.Lat,
				prev.Lon,
				points[i].Lat,
				points[i].Lon,
			)

			totalDistance += segmentDistance

			if points[i].Elevation > prev.Elevation { //add to elevation gain if elevation increases
				totalElevation += points[i].Elevation - prev.Elevation
			}
		}

		points[i].Distance = totalDistance
		points[i].ElevGain = totalElevation
		points[i].Duration = points[i].Time.Sub(startTime)
	}

	//track totals
	first, last := points[0], points[len(points)-1]
	track.Distance = last.Distance
	track.ElevationGain = last.ElevGain
	track.ActivityTime = last.Duration
	track.NetElev = last.Elevation - first.Elevation //net elevation change from start to end
	track.HeartRate = trackHeartRate(points)

	return nil
}

// trackHeartRate returns the time-weighted average heart rate of the whole activity,
// or nil if no points have heart rate data. Calculated the same way as each split's heart rate.
func trackHeartRate(points []TrackPoint) *int {
	//a single point covers no time, so its reading is the average
	if len(points) == 1 {
		if points[0].HeartRate == nil {
			return nil
		}
		hr := *points[0].HeartRate
		return &hr
	}

	seg := 0
	return avgHeartRate(points, byTime, 0, points[len(points)-1].Duration.Seconds(), true, &seg)
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
