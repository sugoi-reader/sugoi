package main

import (
	"fmt"
	"math"
	"os"
	"time"
)

const (
	Nanosecond  time.Duration = 1
	Microsecond               = 1000 * Nanosecond
	Millisecond               = 1000 * Microsecond
	Second                    = 1000 * Millisecond
	Minute                    = 60 * Second
	Hour                      = 60 * Minute
	Day                       = 24 * Hour
	Month                     = 30 * Day
	Year                      = 12 * Month
)

func Days(d time.Duration) float64 {
	hour := d / Day
	nsec := d % Day
	return float64(hour) + float64(nsec)/(60*60*24*1e9)
}

func Months(d time.Duration) float64 {
	hour := d / Month
	nsec := d % Month
	return float64(hour) + float64(nsec)/(60*60*24*30*1e9)
}

func Years(d time.Duration) float64 {
	hour := d / Year
	nsec := d % Year
	return float64(hour) + float64(nsec)/(60*60*24*30*12*1e9)
}

var buildStr string
var buildTime time.Time

func InitializeBuildTime() {
	fileStat, err := os.Stat(os.Args[0])

	if err != nil {
		buildStr = "-"
	}

	buildTime = fileStat.ModTime()
	buildStr = buildTime.Format("2006-01-02 15:04")
}

func BuildDiff() string {
	diff := time.Now().Sub(buildTime)
	return DurationFormat(diff)
}

func DurationFormat(d time.Duration) string {
	var num int
	var sc string
	if d < (Minute) {
		num = int(math.Floor(d.Seconds()))
		if num == 1 {
			sc = "second"
		} else {
			sc = "seconds"
		}
	} else if d < (Hour) {
		num = int(math.Floor(d.Minutes()))
		if num == 1 {
			sc = "minute"
		} else {
			sc = "minutes"
		}
	} else if d < (Day) {
		num = int(math.Floor(d.Hours()))
		if num == 1 {
			sc = "hour"
		} else {
			sc = "hours"
		}
	} else if d < (Month) {
		num = int(math.Floor(Days(d)))
		if num == 1 {
			sc = "day"
		} else {
			sc = "days"
		}
	} else if d < (Year) {
		num = int(math.Floor(Months(d)))
		if num == 1 {
			sc = "month"
		} else {
			sc = "months"
		}
	} else {
		num = int(math.Floor(Years(d)))
		if num == 1 {
			sc = "year"
		} else {
			sc = "years"
		}
	}

	return fmt.Sprintf("%d %s", num, sc)
}
