package route

import (
	"math"
	"motion"
)

func (o *Route) Ride(c *motion.BikeCalc, p par) {

	prexit := max(3, p.Ride.MinSpeed)

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		if s.distHor <= distTOL {
			continue
		}
		s.distLeft = s.dist
		s.vEntry = prexit
		s.vExit = prexit //***

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
		s.brake(c, p)		
		if s.distLeft > 0 && s.powerTarget > 0 && s.vExit > s.vTarget {
			s.decelerate(c, p)
		}
	case s.useConstantVel(c, p):

	case s.vEntry < s.vTarget:
		s.accelerate(c, p)

	case s.vEntry > s.vTarget:
		s.decelerate(c, p)
	}
	if s.distLeft > 0 {
		s.rideConstantVel(c, p)
	}
}

func (s *segment) useConstantVel(c *motion.BikeCalc, p par) bool {
	const (
		maxSeconds = 6
		velTolRel  = 0.075
		velTol     = 0.015
	)
	P := &p.Powermodel

	dv := math.Abs(s.vTarget - s.vExit)
	if dv < velTol {
		return true
	}
	if !p.Ride.KeepEntrySpeed {
		return false
	}
	if dv/s.vTarget > velTolRel ||
		s.powerTarget <= 0 && s.vExit < s.vTarget ||
		s.powerTarget >= P.UphillPower && s.vExit > s.vTarget ||
		s.dist/s.vExit > maxSeconds ||
		s.vExit >= s.vMax {
		return false
	}
	power := c.PowerFromVel(s.vExit)

	if s.vExit < P.MaxPedaledSpeed && power > powerTOL ||
		s.vExit > P.MaxPedaledSpeed && power < -powerTOL {
		return true
	}
	return false
}

func (s *segment) rideConstantVel(c *motion.BikeCalc, p par) {
	var (
		dist      = s.distLeft
		time      = dist / s.vExit
		jouleDrag = -dist * c.Fdrag(s.vExit)
	)
	s.calcSteps += 1
	s.jouleDrag += jouleDrag
	s.distLeft = 0
	s.time += time

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
		s.powerBraking = s.jouleBraking / s.timeBraking
		s.appendPath(brakingConstVel)

	default:
		s.timeFreewheel += time
		s.distFreewheel += dist
		s.jouleDragFreewh += jouleDrag
		s.appendPath(freewheeling)
	}
}

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5 * c.MassKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)

	if s.time > s.timeBraking {
		s.powerRider = s.jouleRider / (s.time - s.timeBraking)
	}
}
