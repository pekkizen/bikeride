package route

import "math"

func (o *Route) SetupRoad(p par) {

	radius := radiusLat(o.LatMean) // in km
	eleFactor := (radius + o.EleMean*m2km) / radius
	o.metersLon = metersLon(o.LatMean) * eleFactor
	o.metersLat = metersLat(o.LatMean) * eleFactor

	// o.metersLon = metersLon(o.LatMean)
	// o.metersLat = metersLat(o.LatMean)

	o.setWind(p.Environment.WindCourse, p.Environment.WindSpeed)
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
		eleUp      float64
		eleDown    float64
		distSum    float64
		distHorSum float64
		r          = o.route
		next       = &r[1]
		s          *segment
		median     = 0.6 * o.distMean
		weight     = 20 * median
		last       = o.segments + 1
	)
	for i := 2; i <= last; i++ {
		s, next = next, &r[i]
		var (
			dLon     = (next.lon - s.lon) * o.metersLon
			dLat     = (next.lat - s.lat) * o.metersLat
			dEle     = next.ele - s.ele
			distHor  = math.Sqrt(dLon*dLon + dLat*dLat)
			distRoad = math.Sqrt(distHor*distHor + dEle*dEle)
		)
		distSum += distRoad
		distHorSum += distHor

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
		if distHor < median {
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
			s.course = course(roadSin, roadCos)
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
		// s.wind = math.Cos(o.windCourse - s.course) * o.windSpeed
	}
	o.eleUp = eleUp
	o.eleDown = -eleDown
	o.eleUpGPX = eleUp
	o.eleDownGPX = -eleDown
	o.distMean = distSum / float64(o.segments)
	o.distMedian = median
	o.distGPX = distSum
	dEle := r[last].ele - r[1].ele
	o.distLine = math.Sqrt(distHorSum*distHorSum + dEle*dEle)
}

// course calculates compass bearing in radians (0 - 2π) from unit direction vector.
// sin = dLon / sqrt(dLat^2 + dLon^2)
// cos = dLat / sqrt(dLat^2 + dLon^2)
// math.Mod(math.Atan2(sin, cos) + 2 * π, 2 * π) gives same to 0.0001, slower 3-4 x
func course(sin, cos float64) float64 {

	a := fastAsin(cos)
	if sin > 0 {
		return (π / 2) - a
	}
	return (3 * π / 2) + a
}

// fastAsin returns the arcus sin of x in radians.
// Handbook of Mathematical Functions, by Milton Abramowitz and Irene Stegun
// page 81, formula 4.4.45, 0 <= x <= 1, error <= 5e-5
func fastAsin(x float64) float64 {
	const (
		a0 = 1.5707288
		a1 = -0.2121144
		a2 = 0.0742610
		a3 = -0.0187293
	)
	neg := false
	if x < 0 {
		x = -x
		neg = true
	}
	x = (π / 2) - math.Sqrt(1-x)*(a0+x*(a1+x*(a2+x*a3)))
	if neg {
		x = -x
	}
	return x
	// branchless: not faster and less inlineable
	// z := math.Abs(x)
	// z = (π / 2) - math.Sqrt(1-z)*(a0+z*(a1+z*(a2+z*a3)))
	// return math.Copysign(z, x)
}

// https://en.wikipedia.org/wiki/Geographic_coordinate_system#Latitude_and_longitude
// https://en.wikipedia.org/wiki/Earth_radius

// metersLat return length of one latitude degree in meters at latitude lat.
func metersLat(lat float64) float64 {
	lat *= deg2rad
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat) //- 0.0023*math.Cos(6*lat)
}

// meterslon return length of one longitude degree in meters at latitude lat.
func metersLon(lat float64) float64 {
	lat *= deg2rad
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) + 0.118*math.Cos(5*lat)
}

// radiusLat returns the geocentric radius of the earth in km at latitude lat.
func radiusLat(lat float64) float64 {
	const (
		R1 = 6378.137 // Equatorial radius
		R2 = 6356.752 // Polar radius
	)
	lat *= deg2rad
	cosLat := math.Cos(lat)
	sinLat := math.Sin(lat)
	a := R1 * R1 * cosLat
	b := R2 * R2 * sinLat
	c := R1 * cosLat
	d := R2 * sinLat
	return math.Sqrt((a*a + b*b) / (c*c + d*d))
}

// angle(a, b) =  math.Abs(math.Mod(a-b + 3*π, 2*π) - π) to 1e-15

// angle returns positive angle (<= π) of two driving courses.
// 0 <= a,b <= 2π
func angle(a, b float64) float64 {

	if a -= b; a < 0 {
		a = -a
	}
	if a < π {
		return a
	}
	return (2 * π) - a
}

// turnRadius approximates turn radius from three consecutive road segments.
func (o *Route) turnRadius() {
	const (
		minRadius = 5.0
		maxRadius = 80
	)
	var (
		r    = o.route
		s    = &r[1]
		next = &r[2]
		prev *segment
		last = o.segments
	)
	_ = r[last]
	for i := 3; i <= last; i++ {
		prev, s, next = s, next, &r[i]

		turnangle := angle(next.course, prev.course)
		turndist := s.distHor + 0.5*(next.distHor+prev.distHor)

		if dist := 3 * s.distHor; dist < turndist {
			turndist = dist
		}

		radius := turndist / turnangle

		if radius > maxRadius {
			// s.radius = 0
			continue
		}
		if radius < minRadius {
			radius = minRadius
		}
		s.radius = radius
	}
}
