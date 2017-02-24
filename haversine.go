package main

import (
	"math"
)

// Mean radius of Earth in kilometers
var Rk = 6371.0

// Mean radius of Earth in miles
var Rm = 3959.0

func rad(deg float64) float64 {
	return deg * math.Pi / 180
}

// haversin(θ) function
func hav(theta float64) float64 {
	return .5 * (1 - math.Cos(theta))
}

func hav2(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// Calculate distance using law of cosines.
// d = acos( sin φ1 ⋅ sin φ2 + cos φ1 ⋅ cos φ2 ⋅ cos Δλ ) ⋅ R
//Values are in radians
func Distance(dlon, dlat, dtlon, dtlat float64) float64 {
	lon := rad(dlon)
	lat := rad(dlat)
	tlon := rad(dtlon)
	tlat := rad(dtlat)

	return Rk *
		math.Acos(
			math.Sin(lat)*math.Sin(tlat)+
				math.Cos(lat)*math.Cos(tlat)*
					math.Cos(lon-tlon))
}

// Calculate distance using Haversine formula. Values are in radians
func Distance2(dlon, dlat, dtlon, dtlat float64) float64 {
	lon := rad(dlon)
	lat := rad(dlat)
	tlon := rad(dtlon)
	tlat := rad(dtlat)

	h := hav(tlat-lat) + math.Cos(lat)*math.Cos(tlat)*hav(tlon-lon)

	return 2 * Rk * math.Asin(math.Sqrt(h))
}
