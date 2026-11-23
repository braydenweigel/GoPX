package gpx_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/braydenweigel/GoPX/gpx"
)

func ExampleParseFile() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(g.Track.Name)
	fmt.Printf("%.2f km\n", g.Track.Distance/1000)
	fmt.Println(g.Track.ActivityTime)
	fmt.Printf("%.1f m climbed\n", g.Track.ElevationGain)
	fmt.Println(*g.Track.HeartRate, "bpm")
	// Output:
	// Shamrock Run 8k
	// 8.05 km
	// 27m24s
	// 13.3 m climbed
	// 175 bpm
}

func ExampleGPX_CalcSplitsDistance() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	// 1km splits for the whole activity, including the leftover distance at the end
	splits, err := g.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, true)
	if err != nil {
		log.Fatal(err)
	}

	for i, s := range splits.Splits {
		fmt.Printf("%d: %.2f km in %v, %d bpm\n", i+1, s.Length, s.Duration.Round(time.Second), *s.HeartRate)
	}
	// Output:
	// 1: 1.00 km in 3m23s, 169 bpm
	// 2: 1.00 km in 3m22s, 168 bpm
	// 3: 1.00 km in 3m26s, 175 bpm
	// 4: 1.00 km in 3m27s, 181 bpm
	// 5: 1.00 km in 3m24s, 177 bpm
	// 6: 1.00 km in 3m29s, 176 bpm
	// 7: 1.00 km in 3m26s, 180 bpm
	// 8: 1.00 km in 3m18s, 174 bpm
	// 9: 0.05 km in 10s, 182 bpm
}

func ExampleGPX_CalcSplitsTime() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	// three 5 minute splits, starting 1 minute in
	splits, err := g.CalcSplitsTime(60, 3, 300, false)
	if err != nil {
		log.Fatal(err)
	}

	for i, s := range splits.Splits {
		fmt.Printf("%d: %.0f m in %v\n", i+1, s.Length, s.Duration)
	}
	// Output:
	// 1: 1521 m in 5m0s
	// 2: 1449 m in 5m0s
	// 3: 1469 m in 5m0s
}

func ExampleGPX_TrimTime() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	// cut the first 2 minutes and the last minute
	trimmed, err := g.TrimTime(120, 60)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(g.Track.ActivityTime, "->", trimmed.Track.ActivityTime)
	fmt.Println(g.StartTime, "->", trimmed.StartTime)
	// Output:
	// 27m24s -> 24m24s
	// 2026-03-15 16:00:00 +0000 UTC -> 2026-03-15 16:02:00 +0000 UTC
}

func ExampleGPX_TrimDistance() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	// cut half a mile from the start
	trimmed, err := g.TrimDistance(gpx.Miles, 0.5, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%.2f km -> %.2f km\n", g.Track.Distance/1000, trimmed.Track.Distance/1000)
	// Output: 8.05 km -> 7.25 km
}

func ExampleGPX_WriteGPX() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	trimmed, err := g.TrimTime(120, 0)
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.MkdirTemp("", "gpx-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "trimmed.gpx")
	if err := trimmed.WriteGPX(path); err != nil {
		log.Fatal(err)
	}

	// the written file can be parsed again
	written, err := gpx.ParseFile(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(written.Track.Name, written.Track.ActivityTime)
	// Output: Shamrock Run 8k 25m24s
}

// Mile splits with pace (time per mile) for each split.
func ExampleGPX_CalcSplitsDistance_pace() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	splits, err := g.CalcSplitsDistance(gpx.Miles, 0, 0, 1, true)
	if err != nil {
		log.Fatal(err)
	}

	for i, s := range splits.Splits {
		pace := time.Duration(float64(s.Duration) / s.Length) //works for the partial last split too
		fmt.Printf("mile %d: %.3f mi, %v/mi pace\n", i+1, s.Length, pace.Round(time.Second))
	}
	// Output:
	// mile 1: 1.000 mi, 5m24s/mi pace
	// mile 2: 1.000 mi, 5m33s/mi pace
	// mile 3: 1.000 mi, 5m29s/mi pace
	// mile 4: 1.000 mi, 5m34s/mi pace
	// mile 5: 1.000 mi, 5m22s/mi pace
	// mile 6: 0.005 mi, 6m5s/mi pace
}

// Splits that skip a warm-up: five 1km splits starting 1km into the activity.
func ExampleGPX_CalcSplitsDistance_skipWarmup() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	splits, err := g.CalcSplitsDistance(gpx.Kilometers, 1, 5, 1, false)
	if err != nil {
		log.Fatal(err)
	}

	var total time.Duration
	for _, s := range splits.Splits {
		total += s.Duration
	}
	fmt.Println(len(splits.Splits), "splits, km 1 to 6 in", total.Round(time.Second))
	// Output: 5 splits, km 1 to 6 in 17m7s
}

// Finding the fastest and slowest 1km of an activity.
func ExampleGPX_CalcSplitsDistance_fastest() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	splits, err := g.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, false) //full splits only, so they compare fairly
	if err != nil {
		log.Fatal(err)
	}

	fastest, slowest := 0, 0
	for i, s := range splits.Splits {
		if s.Duration < splits.Splits[fastest].Duration {
			fastest = i
		}
		if s.Duration > splits.Splits[slowest].Duration {
			slowest = i
		}
	}
	fmt.Printf("fastest: km %d in %v\n", fastest+1, splits.Splits[fastest].Duration.Round(time.Second))
	fmt.Printf("slowest: km %d in %v\n", slowest+1, splits.Splits[slowest].Duration.Round(time.Second))
	// Output:
	// fastest: km 8 in 3m18s
	// slowest: km 6 in 3m29s
}

// Elevation for each split, using quarter-mile splits.
func ExampleGPX_CalcSplitsDistance_elevation() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	splits, err := g.CalcSplitsDistance(gpx.Miles, 0, 4, 0.25, false) //the first mile
	if err != nil {
		log.Fatal(err)
	}

	for i, s := range splits.Splits {
		fmt.Printf("%d: +%.1f m climbed, %+.1f m net\n", i+1, s.ElevGain, s.NetElev)
	}
	// Output:
	// 1: +1.5 m climbed, +0.8 m net
	// 2: +2.9 m climbed, +0.3 m net
	// 3: +0.3 m climbed, -0.1 m net
	// 4: +0.2 m climbed, -0.3 m net
}

// Splits that don't fit return an error explaining why.
func ExampleGPX_CalcSplitsDistance_error() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	_, err = g.CalcSplitsDistance(gpx.Miles, 0, 6, 1, false) //6 miles in an 8km (5 mile) activity
	fmt.Println(err)
	// Output: 6 intervals of 1mi starting at 0mi do not fit in a 5.00mi activity
}

// Heart rate for each minute of the first 5 minutes.
func ExampleGPX_CalcSplitsTime_heartRate() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	splits, err := g.CalcSplitsTime(0, 5, 60, false)
	if err != nil {
		log.Fatal(err)
	}

	for i, s := range splits.Splits {
		if s.HeartRate == nil {
			fmt.Printf("minute %d: no heart rate\n", i+1)
			continue
		}
		fmt.Printf("minute %d: %d bpm, %.0f m\n", i+1, *s.HeartRate, s.Length)
	}
	// Output:
	// minute 1: 170 bpm, 264 m
	// minute 2: 168 bpm, 327 m
	// minute 3: 168 bpm, 296 m
	// minute 4: 168 bpm, 301 m
	// minute 5: 168 bpm, 304 m
}

// 10 minute splits for the whole activity, with and without the leftover time.
func ExampleGPX_CalcSplitsTime_partial() {
	g, err := gpx.ParseFile("testdata/test1.gpx") //27m24s
	if err != nil {
		log.Fatal(err)
	}

	full, err := g.CalcSplitsTime(0, 0, 600, false)
	if err != nil {
		log.Fatal(err)
	}
	withPartial, err := g.CalcSplitsTime(0, 0, 600, true)
	if err != nil {
		log.Fatal(err)
	}

	last := withPartial.Splits[len(withPartial.Splits)-1]
	fmt.Println(len(full.Splits), "full splits")
	fmt.Println(len(withPartial.Splits), "splits with partial, the last one", last.Duration.Round(time.Second))
	// Output:
	// 2 full splits
	// 3 splits with partial, the last one 7m24s
}

// Trimming an activity down to exactly 5km.
func ExampleGPX_TrimDistance_exactDistance() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	extra := g.Track.Distance/1000 - 5 //km past 5km
	trimmed, err := g.TrimDistance(gpx.Kilometers, 0, extra)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%.3f km in %v\n", trimmed.Track.Distance/1000, trimmed.Track.ActivityTime.Round(time.Second))
	// Output: 5.000 km in 17m1s
}

// Removing a warm-up and cool-down, then splitting what's left.
func ExampleGPX_TrimDistance_thenSplits() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	//cut 1km from each end
	main, err := g.TrimDistance(gpx.Kilometers, 1, 1)
	if err != nil {
		log.Fatal(err)
	}

	//splits are measured from the start of the trimmed activity
	splits, err := main.CalcSplitsDistance(gpx.Kilometers, 0, 0, 1, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%.2f km, %d splits, %d bpm average\n", main.Track.Distance/1000, len(splits.Splits), *main.Track.HeartRate)
	// Output: 6.05 km, 7 splits, 176 bpm average
}

// Cutting off time spent at the end of an activity after forgetting to stop recording.
func ExampleGPX_TrimTime_forgotToStop() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	trimmed, err := g.TrimTime(0, 90) //the last 90 seconds
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(trimmed.Track.ActivityTime)
	fmt.Printf("%.2f km -> %.2f km\n", g.Track.Distance/1000, trimmed.Track.Distance/1000)
	// Output:
	// 25m54s
	// 8.05 km -> 7.59 km
}

// Trimming more than the activity's length returns an error, and the original is unchanged.
func ExampleGPX_TrimTime_error() {
	g, err := gpx.ParseFile("testdata/test1.gpx")
	if err != nil {
		log.Fatal(err)
	}

	_, err = g.TrimTime(1000, 1000) //2000s from a 1644s activity
	fmt.Println(err)
	fmt.Println(g.Track.ActivityTime)
	// Output:
	// trimming 1000s from the start and 1000s from the end leaves nothing of a 1644s activity
	// 27m24s
}
