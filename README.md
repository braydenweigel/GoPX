# GoPX

A Go package for reading, analyzing, trimming, and writing GPX activity files, such as runs and rides exported from Strava or Garmin.

- **Parse** a GPX file and get its distance, time, elevation gain, net elevation change, and average heart rate
- **Split** an activity by distance (every mile, every km) or by time (every 5 minutes)
- **Trim** distance or time from the start or end of an activity
- **Write** an activity back to a GPX file that other apps can import, heart rate included

## Install

```sh
go get github.com/braydenweigel/GoPX/gpx
```

## Usage

### Parse a file

```go
package main

import (
	"fmt"
	"log"

	"github.com/braydenweigel/GoPX/gpx"
)

func main() {
	g, err := gpx.ParseFile("run.gpx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(g.Track.Name)                       // Shamrock Run 8k
	fmt.Printf("%.2f km\n", g.Track.Distance/1000)  // 8.05 km
	fmt.Println(g.Track.ActivityTime)               // 27m24s
	fmt.Printf("%.1f m\n", g.Track.ElevationGain)   // 13.3 m
	fmt.Printf("%.1f m\n", g.Track.NetElev)         // 0.3 m
	if g.Track.HeartRate != nil {                   // nil if the file has no heart rate data
		fmt.Println(*g.Track.HeartRate, "bpm")      // 175 bpm
	}
}
```

### Splits

```go
// 1km splits for the whole activity, plus the leftover distance as a final split
splits, err := g.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, true)

for i, s := range splits.Splits {
	fmt.Printf("%d: %.2f km in %v\n", i+1, s.Length, s.Duration)
}
// 1: 1.00 km in 3m23s
// ...
// 8: 1.00 km in 3m18s
// 9: 0.05 km in 10s
```

```go
// three 5 minute splits, starting 1 minute in
splits, err := g.CalcSplitsTime(60, 3, 300, false)
```

Both functions take the same arguments, in the units of the split:

| Argument | Meaning |
|---|---|
| `start` | Where the first split begins. `0` is the start of the activity. |
| `intervals` | How many splits. `0` means as many full splits as fit. |
| `length` | The length of each split. |
| `partial` | When `intervals` is `0`, add whatever is left over as a final, shorter split. Ignored otherwise. |

`CalcSplitsDistance` also takes a unit (`gpx.Miles`, `gpx.Kilometers`, or `gpx.Feet`). `CalcSplitsTime` uses seconds.

Each `Split` has its `Length`, `Duration`, `StartPoint` and `EndPoint` coordinates, `ElevGain`, `NetElev`, and average `HeartRate`. For time splits, `Length` is the distance covered in meters.

### Trim

Trimming returns a new activity with all statistics recalculated. The original isn't changed.

```go
trimmed, err := g.TrimTime(120, 60)                    // cut the first 2 minutes and the last minute
trimmed, err := g.TrimDistance(gpx.Miles, 0.5, 0)     // cut the first half mile
```

### Write

```go
err := trimmed.WriteGPX("trimmed.gpx")
```

Writes a GPX 1.1 file. Heart rate is written in Garmin's `TrackPointExtension` format, which Strava and Garmin Connect read. An existing file at the path is overwritten.

### Example Usages

**Pace per mile**

```go
splits, err := g.CalcSplitsDistance(gpx.Miles, 0, 0, 1, true)
for i, s := range splits.Splits {
	pace := time.Duration(float64(s.Duration) / s.Length) // also correct for the shorter last split
	fmt.Printf("mile %d: %v/mi\n", i+1, pace.Round(time.Second))
}
// mile 1: 5m24s/mi
// mile 2: 5m33s/mi
// ...
```

**Fastest kilometer**

```go
splits, err := g.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, false) // full splits only, so they compare fairly
fastest := 0
for i, s := range splits.Splits {
	if s.Duration < splits.Splits[fastest].Duration {
		fastest = i
	}
}
// fastest: km 8 in 3m18s
```

**Skip a warm-up**: five 1km splits starting 1km in

```go
splits, err := g.CalcSplitsDistance(gpx.Kilometers, 1, 5, 1, false)
```

**Elevation per quarter mile** over the first mile

```go
splits, err := g.CalcSplitsDistance(gpx.Miles, 0, 4, 0.25, false)
for _, s := range splits.Splits {
	fmt.Printf("+%.1f m climbed, %+.1f m net\n", s.ElevGain, s.NetElev)
}
```

**Heart rate each minute**

```go
splits, err := g.CalcSplitsTime(0, 5, 60, false) // the first 5 minutes
for i, s := range splits.Splits {
	if s.HeartRate != nil {
		fmt.Printf("minute %d: %d bpm\n", i+1, *s.HeartRate)
	}
}
```

**Trim to exactly 5km**

```go
extra := g.Track.Distance/1000 - 5
fiveK, err := g.TrimDistance(gpx.Kilometers, 0, extra)
// 5.000 km in 17m1s
```

**Forgot to stop recording**: cut the last 90 seconds and save

```go
fixed, err := g.TrimTime(0, 90)
if err != nil {
	log.Fatal(err)
}
err = fixed.WriteGPX("fixed.gpx")
```

**Remove a warm-up and cool-down, then split what's left**

```go
main, err := g.TrimDistance(gpx.Kilometers, 1, 1)
splits, err := main.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, true) // measured from the trimmed start
```

**Errors explain what's wrong**

```go
_, err := g.CalcSplitsDistance(gpx.Miles, 0, 6, 1, false)
fmt.Println(err) // 6 intervals of 1mi starting at 0mi do not fit in a 5.00mi activity

_, err = g.TrimTime(1000, 1000)
fmt.Println(err) // trimming 1000s from the start and 1000s from the end leaves nothing of a 1644s activity
```

The full versions of these are runnable examples in [gpx/example_test.go](gpx/example_test.go), and they show up in `go doc`.

## Units

| Value | Stored as |
|---|---|
| Distance | meters (`float64`) |
| Elevation | meters (`float64`) |
| Time | `time.Duration` |
| Heart rate | beats per minute (`*int`, `nil` when not recorded) |

Function inputs use the unit you pass for distances, and seconds (`float64`) for times.

## How values are calculated

- **Distance** between points uses the haversine formula on a spherical Earth.
- **Elevation gain** adds up every increase between consecutive points. Descents don't subtract from it. **Net elevation** is the end elevation minus the start elevation.
- **Heart rate** averages are weighted by time, so a reading that applied for 10 seconds counts ten times as much as one that applied for 1 second.
- **Split and trim points** rarely land exactly on a recorded point, so position, time, elevation, and heart rate there are estimated from the points on either side.

## Limitations

- Reads one track with one segment per file, which is the format Strava and Garmin export.
- Waypoints, routes, and extensions other than heart rate are ignored, and aren't written back out.

## Documentation

Full API documentation, with runnable examples:

```sh
go doc -all github.com/braydenweigel/GoPX/gpx
```

## Development

```sh
cd gpx
go test ./...
```

On Windows with Smart App Control turned on, Go programs may be blocked from running. Run the tests from WSL instead.
