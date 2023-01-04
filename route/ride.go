package route

import (
	"bikeride/motion"
	"math"
	// "fmt"
)

func (o *Route) Ride(c *motion.BikeCalc, p par) {

	prexit := max(3, p.Ride.MinSpeed)

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		s.distLeft = s.dist
		s.vEntry = prexit
		s.vExit = prexit //!
		if s.distHor <= distTOL {
			continue
		}
		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		s.ride(c, p)

		prexit = s.vExit
		s.calcJoules(c)
		o.time += s.time
		o.jouleRider += s.jouleRider
	}
}

func (s *segment) ride(c *motion.BikeCalc, p par) {

	switch {
	case s.vEntry > s.vMax:
		if !s.brake(c, p) {
			s.decelerate(c, p)
		}
	case s.useConstantVel(p):

	case s.vEntry < s.vTarget:
		s.accelerate(c, p)

	case s.vEntry > s.vTarget:
		s.decelerate(c, p)
	}
	if s.distLeft > 0 {
		s.rideConstantVel(c, p)
	}
}

func (s *segment) useConstantVel(p par) bool {

	return math.Abs(s.vTarget-s.vEntry) < 0.1*velTOL
}

func (s *segment) rideConstantVel(c *motion.BikeCalc, p par) {

	dist := s.distLeft
	s.distLeft = 0
	time := dist / s.vExit
	s.time += time
	jouleDrag := -dist * c.Fdrag(s.vExit)
	s.jouleDrag += jouleDrag

	power := c.PowerFromVel(s.vExit)

	switch {
	case power > powerTOL:
		s.jouleRider += time * power
		s.timeRider += time
		s.jouleDragRider += jouleDrag
		s.appendPath(ridingConstVel)

	case power < -powerTOL:
		s.jouleBraking += time * power
		s.timeBraking += time
		s.distBraking += dist
		s.jouleDragBrake += jouleDrag
		s.appendPath(brakingConstVel)

	default:
		s.timeFreewheel += time
		s.distFreewheel += dist
		s.jouleDragFreewheel += jouleDrag
		s.appendPath(freewheeling)
	}
}

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5 * c.MassKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)

	s.powerRider = s.jouleRider / s.time

	// if s.timeFreewheel > 0 && s.jouleRider == 0 {
	// 	s.appendPath(freewheeling)
	// }
}
