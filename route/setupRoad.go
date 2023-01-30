package route

import "math"

// import "fmt"

func (o *Route) SetupRoad(p par) {

	o.setWind(p.WindCourse, p.WindSpeed)
	o.metersLon = metersLon(o.LatMean)
	o.metersLat = metersLat(o.LatMean)
	o.setupSegments()
	if o.limitTurnSpeeds {
		o.turnRadius()
	}
}

func (o *Route) setWind(course, speed float64) {
	o.windSpeed = speed // m/s
	if course < 0 {
		o.windCourse = course
		return
	}
	o.windCourse = course * deg2rad
	// unit wind direction vector.
	o.windSin = math.Sin(o.windCourse)
	o.windCos = math.Cos(o.windCourse)
}

func (o *Route) setupSegments() {

	var (
		eleUp   float64
		eleDown float64
		distTot float64
		r       = o.route
		next    = &r[1]
		median  = 0.6 * o.distMean
		weight  = 20 * median
	)
	for i := 2; i <= o.segments+1; i++ {
		s := next
		next = &r[i]

		dLon := (next.lon - s.lon) * o.metersLon
		dLat := (next.lat - s.lat) * o.metersLat
		distHor := math.Sqrt(dLon*dLon + dLat*dLat)
		if distHor < distTOL {
			distHor = distTOL
		}
		dEle := next.ele - s.ele
		distRoad := math.Sqrt(distHor*distHor + dEle*dEle)
		distTot += distRoad

		s.dist = distRoad
		s.distHor = distHor
		s.grade = dEle / distHor

		if dEle < 0 {
			eleDown += dEle
		} else {
			eleUp += dEle
		}
		//approx. median
		eta := median / weight
		if distRoad < median {
			eta = -eta
		}
		median += eta
		weight++

		if !o.limitTurnSpeeds && o.windSpeed == 0 {
			continue
		}
		//unit direction vector of the road segment
		roadSin := dLon / distHor
		roadCos := dLat / distHor

		if o.limitTurnSpeeds {
			s.course = course(roadCos, roadSin)
		}
		if o.windSpeed == 0 {
			continue
		}
		if o.windCourse < 0 { // constant direct head or tailwind
			s.wind = o.windSpeed
			continue
		}
		// wind component of riding direction
		s.wind = (roadSin*o.windSin + roadCos*o.windCos) * o.windSpeed
		// same by trig
		// s.wind = math.Cos( o.windCourse - s.course) * o.windSpeed
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
// math.Mod(math.Atan2(sin, cos) + 2 * π, 2 * π) gives same to 0.0001, slower 3-4 x
func course(cos, sin float64) float64 {

	a := asin(cos)
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

// https://en.wikipedia.org/wiki/Geographic_coordinate_system#Latitude_and_longitude
// https://msi.nga.mil/Calc

// metersLat return length of one latitude degree in meters
func metersLat(lat float64) float64 {
	lat *= deg2rad
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat) //- 0.0023 * math.Cos(6*lat)
}

// meterslon return length of one longitudedegree in meters
func metersLon(lat float64) float64 {
	lat *= deg2rad
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) //+ 0.118*math.Cos(5*lat)
}

//	angle returns positive angle (<= π) of two driving courses.
//
// angle(a, b) =  math.Abs( math.Mod(a-b + 3*π, 2*π) - π) to 14 desimals
func angle(a, b float64) float64 {
	a -= b
	if a < 0 {
		a = -a
	}
	if a > π {
		return 2*π - a
	}
	return a
}

// turnRadius approximates turn radius from three consecutive road segments
func (o *Route) turnRadius() {
	const (
		minTurn   = π / 10.0 // 18 deg
		minRadius = 6.0
		maxRadius = 100
	)
	var (
		r    = o.route
		s    = &r[1]
		next = &r[2]
	)
	for i := 3; i <= o.segments; i++ {
		prev := s
		s, next = next, &r[i]

		turn := angle(next.course, prev.course)
		if turn < minTurn { 
			continue
		}
		dist := s.distHor
		dist += 0.5 * (min(dist, prev.distHor) + min(dist, next.distHor))
		radius := dist / turn

		if radius < minRadius {
			s.radius = minRadius
			continue
		}
		if radius < maxRadius {
			s.radius = radius
		}
	}
}
