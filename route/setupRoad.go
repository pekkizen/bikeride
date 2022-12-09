package route

import (
	"math"
)

func (o *Route) SetupRoad(p par) {

	o.setWind(p.WindCourse, p.WindSpeed)
	o.metersLon = metersLon(o.LatMean)
	o.metersLat = metersLat(o.LatMean)

	o.setupSegments()
	if o.filter.rounds > 0 {
		o.calcNonFilterable()
	}
	if p.LimitTurnSpeeds {
		o.turnRadius()
	}
}

func (o *Route) setWind(course, speed float64) {
	o.windSpeed = speed // m/s
	if course < 0 {
		o.windCourse = course
		return
	}
	o.windCourse = deg2rad * course
	// Unit direction vector. windCourse = 0 <-> wind from north
	o.windSin = math.Sin(o.windCourse)
	o.windCos = math.Cos(o.windCourse)
}

// This tries to sum elevation of longer ascends.
func (o *Route) calcNonFilterable() {

	dist, dEleSum := 0.0, 0.0
	r := o.route

	for i := 1; i <= o.segments-1; i++ {

		dEle := r[i+1].eleGPX - r[i].eleGPX
		dEleSum += dEle
		dist += r[i].dist

		if dist > 500 && dEleSum > 45 {
			o.nonFilterable += dEleSum
			dEleSum = 0
			dist = 0
		}
		if dist > 500 && dEleSum < -45 {
			dEleSum = 0
			dist = 0
		}
	}

}

func (o *Route) setupSegments() {

	var eleUp, eleDown, distTot float64
	r := o.route
	next := &r[1]
	median := 0.6 * o.distMean
	weight := max(100, 5.0 * o.distMean)

	for i := 2; i <= o.segments+1; i++ {
		s := next
		next = &r[i]

		dLon := (next.lon - s.lon) * o.metersLon
		dLat := (next.lat - s.lat) * o.metersLat
		distHor := math.Sqrt(dLon*dLon + dLat*dLat)

		dEle := next.ele - s.ele
		dist := math.Sqrt(distHor*distHor + dEle*dEle)
		distTot += dist

		s.dist = dist
		s.grade = dEle / distHor
		s.distHor = distHor

		if dEle < 0 {
			eleDown += dEle
		} else {
			eleUp += dEle
		}
		// Approx. median
		// eta := 0.5 * max(-2, min(2, dist-median))*median/ weight
		eta := median/ weight
		if dist < median {
			eta = -eta
		}
		median += eta 
		weight += median / 20

		if !o.limitTurnSpeeds && o.windSpeed == 0 {
			continue
		}

		//unit direction vector of the road segment
		sin := dLon / distHor // y-axle sin
		cos := dLat / distHor
		// wind component of riding direction
		// dot product of unit direction vectors x wind speeed
		s.wind = (sin*o.windSin + cos*o.windCos) * o.windSpeed
		if o.windCourse < 0 { // constant direct headwind or tailwind
			s.wind = o.windSpeed
		}
		if o.limitTurnSpeeds {
			s.course = course(cos, sin)
		}	
	}
	o.eleUp = eleUp
	o.eleDown = -eleDown
	o.eleUpGPX = eleUp
	o.eleDownGPX = -eleDown
	o.distMean = distTot / float64(o.segments)
	o.distMedian = median
}

// sin = dLon / sqrt(dLat^2 + dLon^2)
// cos = dLat / sqrt(dLat^2 + dLon^2)
// y angle cos is x angle sin
// return math.Mod(math.Atan2(sin, cos) + 2 * π, 2 * π) // gives ~same, 3.5 x slower
func course(cos, sin float64) float64 {

	a := asin(cos) // a is x-axle angle
	if sin > 0 {
		return π/2 - a
	}
	return 3*π/2 + a
}

// Handbook of Mathematical Functions, by Milton Abramowitz and Irene Stegun
// page 81, formula 4.4.45, 0 <= x <= 1, error <= 5e-5
func asin(x float64) float64 {

	const (
		a0 = 1.5707288
		a1 = -0.2121144
		a2 = 0.0742610
		a3 = -0.0187293
	)
	neg := x < 0
	if neg {
		x = -x
	}
	x = π/2 - math.Sqrt(1-x)*(a0+x*(a1+x*(a2+x*a3)))
	if neg {
		return -x
	}
	return x
}

// turnRadius approximates radius from four consecutive routepoints
func (o *Route) turnRadius() {
	const (
		minAngle       = π / 12.0 // 15 deg
		maxTurnLimDist = 50.0
		minRadius      = 4.0
		maxRadius      = 100
		maxGrade       = 0.0
	)
	r := o.route
	seg := &r[1]
	next := &r[2]

	seg.radius = 0 // no speed limit
	next.radius = 0

	for i := 3; i <= o.segments; i++ {
		prev := seg
		seg = next
		next = &r[i]
		next.radius = 0

		if seg.grade >= maxGrade {
			continue // downhills only
		}
		dist := seg.dist
		if dist >= maxTurnLimDist {
			continue
		}
		a := angle(next.course - prev.course)
		if a < minAngle {
			continue
		}
		dist += 0.5 * (min(dist, prev.dist) + min(dist, next.dist))
		radius := dist / a

		if radius < minRadius {
			seg.radius = minRadius
		} else if radius > maxRadius {
			seg.radius = radius
		}
	}

}

// angle returns positive angle (<= π) of two driving courses.
// a (-2π <= a <= 2π) is difference of two driving courses (0 <= course <= 2π)
// angle(a) == Abs(Mod(a + 3*π, 2*π) - π)
func angle(a float64) float64 {
	// return 	math.Abs(math.Mod(a + 3*π, 2*π) - π)
	if a < 0 {
		a = -a
	}
	if a > π {
		return 2*π - a
	}
	return a
}

// https://msi.nga.mil/Calc
//  Each calculator page is a self-contained JavaScript program that
//  can be downloaded and used off-line.
//  Set up "Constants"
// 	m1 = 111132.92;		// latitude calculation term 1
// 	m2 = -559.82;		// latitude calculation term 2
// 	m3 = 1.175;			// latitude calculation term 3
// 	m4 = -0.0023;		// latitude calculation term 4
// 	p1 = 111412.84;		// longitude calculation term 1
// 	p2 = -93.5;			// longitude calculation term 2
// 	p3 = 0.118;			// longitude calculation term 3

//  Calculate the length of a degree of latitude and longitude in meters
// 	latlen = m1 + (m2 * Math.cos(2 * lat)) + (m3 * Math.cos(4 * lat)) +
// 		(m4 * Math.cos(6 * lat));
// 	longlen = (p1 * Math.cos(lat)) + (p2 * Math.cos(3 * lat)) +
// 		(p3 * Math.cos(5 * lat));

func metersLat(lat float64) float64 {
	lat *= deg2rad
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat) //- 0.0023 * math.Cos(6*lat)
}
func metersLon(lat float64) float64 {
	lat *= deg2rad
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) + 0.118*math.Cos(5*lat)
}

// func courseAtan(cos, sin float64) float64 {
// 	//Atan(±Inf) = ±Pi/2
// 	a := math.Atan(sin / cos)
// 	if cos < 0 {
// 		return π + a
// 	}
// 	if sin >= 0 {
// 		return a
// 	}
// 	return 2 * π + a
// }

// // courseAtan2 is the "standard" way to calculate course. Not used.
// func courseAtan2(cos, sin float64) float64 {
// 	// return math.Mod(math.Atan2(sin, cos) + 2 * π, 2 * π)
// 	a := math.Atan2(sin, cos)
// 	if a >= 0 {
// 		return a
// 	}
// 	return 2 * π + a
// }
