package route

import "math"

// import "fmt"

func (o *Route) SetupRoad(p par) {

	o.setWind(p.WindCourse, p.WindSpeed)
	o.metersLon = metersLon(o.LatMean)
	o.metersLat = metersLat(o.LatMean)

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
	o.windCourse = course * deg2rad
	// unit wind direction vector.
	o.windSin = math.Sin(o.windCourse)
	o.windCos = math.Cos(o.windCourse)
}

func (o *Route) setupSegments() {

	var (
		eleUp, eleDown float64
		distTot        float64
		r              = o.route
		next           = &r[1]
		median         = 0.6 * o.distMean
		weight         = 20 * median
	)
	for i := 2; i <= o.segments+1; i++ {
		s := next
		next = &r[i]

		dLon := (next.lon - s.lon) * o.metersLon
		dLat := (next.lat - s.lat) * o.metersLat
		distHor := math.Sqrt(dLon*dLon + dLat*dLat)

		dEle := next.ele - s.ele
		distRoad := math.Sqrt(distHor*distHor + dEle*dEle)
		distTot += distRoad

		s.dist = distRoad
		s.grade = dEle / distHor
		s.distHor = distHor

		if dEle < 0 {
			eleDown += dEle
		} else {
			eleUp += dEle
		}
		// approx. median
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
		sin := dLon / distHor
		cos := dLat / distHor

		if o.limitTurnSpeeds {
			s.course = course(cos, sin)
		}
		if o.windSpeed == 0 {
			continue
		}
		if o.windCourse < 0 { // constant direct head or tailwind
			s.wind = o.windSpeed
			continue
		}
		// wind component of riding direction
		s.wind = (sin*o.windSin + cos*o.windCos) * o.windSpeed
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
// y angle cos is x angle sin
// return math.Mod(math.Atan2(sin, cos) + 2 * π, 2 * π) // gives same within 0.0001, 3.5 x slower
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

// turnRadius approximates turn radius from four consecutive routepoints
func (o *Route) turnRadius() {
	const (
		minTurn        = π / 12.0 // 15 deg
		maxTurnLimDist = 50.0
		minRadius      = 5.0
		maxRadius      = 100
		maxGrade       = 0.0
	)
	r := o.route
	s := &r[1]
	next := &r[2]

	s.radius = 0 // no speed limit
	next.radius = 0

	for i := 3; i <= o.segments; i++ {
		prev := s
		s = next
		next = &r[i]
		next.radius = 0

		if s.grade >= maxGrade {
			continue // downhills only
		}
		dist := s.dist
		if dist >= maxTurnLimDist {
			continue
		}
		// turn := TurnAngle(next.course, prev.course)
		turn := angle(next.course - prev.course)
		if turn < minTurn {
			continue
		}
		
		dist += 0.5 * (min(dist, prev.dist) + min(dist, next.dist)) // **************************?
		radius := dist / turn
		// if s.segnum == 1277 {
		// 	x := 180 / π
		// 	fmt.Println("prev", x*prev.course, "s", x*s.course, "next", x*next.course)
		// 	fmt.Println("prev", prev.dist, "s", s.dist, "next", next.dist)
		// 	fmt.Println("dist", dist, "turn", turn, "radius", radius)
		// }


		if radius < minRadius {
			s.radius = minRadius
			continue
		}
		if radius < maxRadius {
			s.radius = radius
		}
	}

}

func TurnAngle(a, b float64) float64 {
	if a > π {
		a -= 2 * π
	}
	if b > π {
		b -= 2 * π
	}
	a = math.Abs(a - b)
	if a > π {
		a = 2 * π - a
	}
	return a
}

// angle returns positive angle (<= π) of two driving courses.
// a (-2π <= a <= 2π) is difference of two driving courses (0 <= course <= 2π)
// angle(a) =  math.Abs( math.Mod(a + 3*π, 2*π) - π) to 14 desimals
func angle(a float64) float64 {
	if a < 0 {
		a = -a
	}
	if a > π {
		return 2*π - a
	}
	return a
}

// https://en.wikipedia.org/wiki/Geographic_coordinate_system#Latitude_and_longitude
// https://msi.nga.mil/Calc
// These functions return length of one lon/latitude degree in meters
func metersLat(lat float64) float64 {
	lat *= deg2rad
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat) //- 0.0023 * math.Cos(6*lat)
}
func metersLon(lat float64) float64 {
	lat *= deg2rad
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) //+ 0.118*math.Cos(5*lat)
}
