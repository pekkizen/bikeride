package route

import (
	"math"
)

func (o *Route) SetupRoad(p par) {
	const radius = 6371.0 // Earth mean radius in km
	// radius := earthRadiusByLatitude(o.LatMean)

	// At 1000 m elevation, the distance is 6372/6371 = 1.000157 times the
	// distance at sea level. This is 15.7 meters / 100 km, which is
	// insignificant compared to the distance error produced by elevation
	// measurement noise in most route data. But we get it for practically free.

	eleCorrection := (radius + o.EleMean/1000) / radius
	o.metersLon = metersLon(o.LatMean) * eleCorrection
	o.metersLat = metersLat(o.LatMean) * eleCorrection

	o.setWind(p.Environment.WindCourse, p.Environment.WindSpeed)
	o.setupSegments()
	if p.Ride.LimitTurnSpeeds {
		o.turnRadius()
	}
}

func (o *Route) setWind(course, speed float64) {
	o.windSpeed = speed // m/s
	if course < 0 {
		o.windCourse = course
		return
	}
	o.windCourse = course * (π / 180)
	o.windSin = math.Sin(o.windCourse) // unit wind direction vectors.
	o.windCos = math.Cos(o.windCourse)
}

func (o *Route) setupSegments() {
	var (
		eleSum     [2]float64
		distSum    float64
		distHorSum float64
		next       = &o.route[1]
		s          *segment
		median     = 0.6 * o.distMean
		weight     = 5.0 * median
		windSpeed  = o.windSpeed
	)
	for i := 2; i < len(o.route); i++ {
		s, next = next, &o.route[i]
		var (
			dLon     = (next.lon - s.lon) * o.metersLon
			dLat     = (next.lat - s.lat) * o.metersLat
			dEle     = next.ele - s.ele
			distHor  = math.Sqrt(dLon*dLon + dLat*dLat)
			distRoad = math.Sqrt(dLon*dLon + dLat*dLat + dEle*dEle)
		)
		eleSum[signbit(dEle)] += dEle                     // Elevation gain/loss
		median += copysign(median/weight, distHor-median) // Approx. road segment length median
		weight++
		s.course = course(dLon, dLat)
		s.grade = dEle / distHor
		s.dist = distRoad
		s.distHor = distHor
		distSum += distRoad
		distHorSum += distHor
		switch {
		case windSpeed == 0:
		case o.windCourse >= 0: // Wind component of riding direction.
			s.wind = windSpeed / distHor * (dLon*o.windSin + dLat*o.windCos)
			// s.wind = math.Cos(o.windCourse-s.course) * windSpeed // by trig
		case o.windCourse == -1: // Constant wind, head or tail
			s.wind = windSpeed
		case o.windCourse == -2: // Constant head or tailwind changing every 31 segments
			if i&31 == 0 {
				windSpeed = -windSpeed
			}
			s.wind = windSpeed
		}

	}
	o.eleUp = eleSum[0]
	o.eleDown = -eleSum[1]
	o.eleUpGPX = o.eleUp
	o.eleDownGPX = o.eleDown
	o.distMean = distSum / float64(o.segments)
	if o.segments < 50 {
		median = 0.6 * o.distMean
	}
	o.distMedian = median
	o.distGPX = distSum
	dEle := o.route[o.segments+1].ele - o.route[1].ele
	o.distLine = math.Sqrt(distHorSum*distHorSum + dEle*dEle)
}

/*
course calculates compass bearing in radians [0 - 2π)
from delta longitude and latitude of two coordinate points.
FULL QUADRANT APPROXIMATIONS FOR THE ARCTANGENT FUNCTION
Xavier Girones, Carme Juli´a and Domenec Puig, 2013.
Formula 18: arctan (︁𝑦/𝑥)︁ ≈ π/2 · (B𝑥𝑦 + 𝑦^2) / (𝑥^2 + 2B𝑥𝑦 + 𝑦^2). [0, π/2], (x>0 || y>0)
Error <= 0.00283 rad, 0.162 deg.
math.Mod(math.Atan2(dLon, dLat)+2*π, 2*π) gives more exact value but is 20 x slower.
Expected 0 < dLon^2 + dLat^2 < +Inf. Non zero finite move to get a proper course.
*/
func course(dLon, dLat float64) float64 {
	const B = 0.596227

	p := math.Abs(B * dLon * dLat)
	q := p + dLon*dLon
	p += dLat * dLat
	atan := (π / 2) * q / (p + q) // atan(abs(dLon/dLat)) in [0, π/2]
	if dLon >= 0 {
		if dLat > 0 {
			return atan
		}
		return π - atan
	}
	if dLat < 0 {
		return π + atan
	}
	return (2 * π) - atan
}

// metersLat return length of one latitude degree in meters at latitude lat. ~100 km
// https://en.wikipedia.org/wiki/Geographic_coordinate_system#Latitude_and_longitude
func metersLat(lat float64) float64 {
	lat *= (π / 180)
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat)
}

// meterslon return length of one longitude degree in meters at latitude lat.
func metersLon(lat float64) float64 {
	lat *= (π / 180)
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) + 0.118*math.Cos(5*lat)
}

// radius = turn distance / turn angle
func (o *Route) turnRadius() {
	var (
		r    = o.route
		s    = &r[1]
		next = &r[2]
		prev *segment
	)
	for i := 3; i < len(r)-1; i++ {
		prev, s, next = s, next, &r[i]

		radius := s.distHor * 2 / angle(prev.course, next.course)
		s.radius = min(max(radius, minRadius), noLimRadius)
	}
}

// func angle(a, b float64) float64 { return math.Abs(math.Mod(a-b+3*π, 2*π) - π) }

// angle returns positive angle (<= π) of two driving courses.
// 0 <= a,b <= 2π
func angle(a, b float64) float64 {
	a = math.Abs(a - b)
	if a < π {
		return a
	}
	return (2 * π) - a
}

// earthRadius returns the geocentric radius of the earth in km at latitude lat.
// https://en.wikipedia.org/wiki/Earth_radius
// func earthRadiusByLatitude(lat float64) float64 {
// 	const (
// 		R1 = 6378.137 // Equatorial radius
// 		R2 = 6356.752 // Polar radius
// 	)
// 	lat *= (π / 180)
// 	c := R1 * math.Cos(lat)
// 	d := R2 * math.Sin(lat)
// 	a := R1 * c
// 	b := R2 * d
// 	return math.Sqrt((a*a + b*b) / (c*c + d*d))
// }

/*
// courseAsin calculates compass bearing in radians (0 - 2π)
// from a unit direction vector.
// sin = dLon / sqrt(dLat^2 + dLon^2)
// cos = dLat / sqrt(dLat^2 + dLon^2)
// math.Mod(math.Atan2(dLon, dLat)+2*π, 2*π) gives ~same but slower 3-4 x
func courseAsin(sin, cos float64) float64 {
	a := fastAsin(cos)
	if sin > 0 {
		return (π / 2) - a
	}
	return (3 * π / 2) + a
}

// fastAsin returns the arcus sin of x in radians.
// Handbook of Mathematical Functions, by Milton Abramowitz and Irene Stegun
// page 81, formula 4.4.45, 0 <= x <= 1, error <= 5e-5, actually 6.7e-5.
func fastAsin(x float64) float64 {
	const (
		a0 = +1.5707288
		a1 = -0.2121144
		a2 = +0.0742610
		a3 = -0.0187293
	)
	z := math.Abs(x)
	z = (π / 2) - math.Sqrt(1-z)*(a0+z*(a1+z*(a2+z*a3)))
	if x < 0 {
		return -z
	}
	return z
}
*/
