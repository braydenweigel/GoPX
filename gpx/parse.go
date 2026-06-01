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

	err = calcStats(gpx.Track.Segment.TrackPoints)
	if err != nil {
		return nil, err
	}
	l := len(gpx.Track.Segment.TrackPoints)
	gpx.Track.Distance = gpx.Track.Segment.TrackPoints[l-1].TotalDistance
	gpx.Track.ElevationGain = gpx.Track.Segment.TrackPoints[l-1].TotalElevGain
	gpx.Track.ActivityTime = gpx.Track.Segment.TrackPoints[l-1].TotalTime

	return &gpx, nil
}

func calcStats(points []TrackPoint) error {
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

		points[i].TotalDistance = totalDistance
		points[i].TotalElevGain = totalElevation
		points[i].TotalTime = points[i].Time.Sub(startTime)
	}

	return nil
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
