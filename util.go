package main

import (
	"math/rand"
	"time"
)

// Abs returns the absolute value of x.
//
// Special cases are:
//	Abs(±Inf) = +Inf
//	Abs(NaN) = NaN
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	if x == 0 {
		return 0 // return correctly abs(-0)
	}
	return x
}

// DaysAgo returns the date N days ago, with time set to 8am, in the local timezone.
func DaysAgo(n int) time.Time {
	now := time.Now()

	then := now.Add(time.Duration(-n*60) * time.Hour)
	return time.Date(then.Year(), then.Month(), then.Day(), 8, 0, 0, 0, time.Local)
	// return now.AddDate(0, 0, -n)
}

// CoinFlip does a coin toss, and returns the positive value passed if heads,
// or the negativeof the value passed if tails.
func CoinFlip(val int, rnd *rand.Rand) int {
	// Flip a coin. Even is heads.  Odd is tails.
	heads := rnd.Intn(100)%2 == 0
	if heads {
		if val == 0 {
			return 1
		} else {
			return Abs(val)
		}
	} else {
		if val == 0 {
			return 0
		} else {
			return -Abs(val)
		}
	}
}
