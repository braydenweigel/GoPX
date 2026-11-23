// Package gpx reads, analyzes, trims, and writes GPX activity files, such as runs
// and rides exported from Strava or Garmin.
//
// # Parsing
//
// [ParseFile] reads a GPX file and calculates the activity's statistics. The totals
// are on [Track]:
//
//	g, err := gpx.ParseFile("run.gpx")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(g.Track.Distance)     // meters
//	fmt.Println(g.Track.ActivityTime) // time.Duration
//
// Each [TrackPoint] also holds running totals from the start of the activity
// (distance, duration, and elevation gain).
//
// # Splits
//
// [GPX.CalcSplitsDistance] and [GPX.CalcSplitsTime] divide an activity into equal
// sections, such as mile splits or 10 minute splits, with each split's duration,
// distance, elevation, and heart rate:
//
//	splits, err := g.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, true) // every km, plus the leftover distance
//
// # Trimming
//
// [GPX.TrimDistance] and [GPX.TrimTime] return a copy of the activity with distance or
// time cut from the start and/or end. The original is never modified:
//
//	trimmed, err := g.TrimTime(120, 0) // cut the first 2 minutes
//
// # Writing
//
// [GPX.WriteGPX] writes an activity to a GPX 1.1 file that other apps can import,
// including heart rate data.
//
// # Units
//
// Distances and elevations are stored in meters, and times as [time.Duration].
// Functions that take a distance also take a [Unit] ([Miles], [Kilometers], or [Feet]);
// functions that take a time use seconds as a float64.
//
// # Calculations
//
//   - Distance between points uses the haversine formula on a spherical Earth.
//   - Elevation gain adds up every increase between consecutive points; decreases are ignored.
//   - Heart rate averages are weighted by time: each reading counts for how long it applied,
//     changing linearly between points. Points without a reading use their neighbor's.
//   - Split boundaries and trim points rarely land exactly on a recorded point, so position,
//     time, elevation, and heart rate there are interpolated between the points on either side.
//
// The package reads one track with one segment per file, which is what Strava and Garmin export.
package gpx
