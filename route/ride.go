package route

import (
	"bikeride/motion"
	"bikeride/logerr"
	"math"
)

func (o *Route) Ride(c *motion.BikeCalc, p par, l *logerr.Logerr) {

	// prexit := p.Powermodel.FlatSpeed
	prexit := p.MinSpeed

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		s.distLeft = s.dist
		s.vEntry = prexit
		s.vExit = prexit 
		if s.distHor <= distTol {
			continue
		}
		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		s.ride(c, p, l)

		prexit = s.vExit
		s.calcJoules(c)
		o.time += s.time
		o.jouleRider += s.jouleRider
		if deBUG {
			s.checkRide(c, p, l)
		}
	}
}

func (s *segment) checkRide(c *motion.BikeCalc, p par, l *logerr.Logerr)  {

	freewheelBraking := s.calcPath == 30
	braking := s.calcPath == 2 || s.calcPath == 29 || s.calcPath == 295
	n := s.segnum

	if s.powerBraking < 0 && s.powerTarget > p.Powermodel.FlatPower {
		l.Msg(0,"seg", n,"powerBraking < 0 && powerTarget > FlatPower")
	}
	if s.vExit > s.vMax && !braking && !freewheelBraking {
		l.Msg(0,"seg", n,"vExit > vMax")
	}
	if s.vExit < p.MinSpeed { 
		l.Msg(0,"seg", n, "vExit < MinSpeed")
	}
	if s.powerRider > p.PowerAcceCoef * s.powerTarget + 0.1 && 
		s.vTarget >  p.MinSpeed && s.powerTarget > p.PowerAcceMinPros * p.Powermodel.FlatPower {
		l.Msg(0,"seg", n,"powerRider > PowerAcceCoef * powerTarget")
	}
}

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5*c.MassKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)
	
	// Rider power is taken over all non braking time.
	timeRider := s.time
	if s.timeBraking > 0 {
		s.powerBraking = s.jouleBraking / s.timeBraking
		timeRider -= s.timeBraking
	}
	if timeRider > 0 && s.jouleRider > 0 {
		s.powerRider = s.jouleRider / timeRider
	}
}

func (s *segment) useConstantVel(p par) bool {

	if s.vTarget == s.vEntry {
		return true
	}
	if s.powerTarget == 0 {
		return false
	}
	if math.Abs(s.vTarget - s.vEntry) < sameVelTol {
		return true
	}
	return false
}

func (s *segment) ride(c *motion.BikeCalc, p par, l *logerr.Logerr) {

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
		s.rideConstantVel(c, p, l)
	}
}

func (s *segment) rideConstantVel(c *motion.BikeCalc, p par, l *logerr.Logerr) {
	if deBUG {
		brakesHold := s.calcPath != 29
		if math.Abs(s.vExit - s.vTarget) > 16*sameVelTol && brakesHold {
			l.Msg(0, "rideConstantVel: seg", s.segnum, "abs(vExit - vTarget) > sameVelTol", s.calcPath)
		}
	}
	
	dist := s.distLeft
	s.distLeft = 0
	time := dist / s.vExit
	s.time += time
	s.jouleDrag -= dist * c.Fdrag(s.vExit)

	power := c.PowerFromVel(s.vExit)

	switch {
	case power > p.MinPower:
		s.jouleRider += time * power
		s.timeRider += time
		s.appendPath(ridingConstVel)
	case power < -p.MinPower:
		s.jouleBraking += time * power
		s.timeBraking += time
		s.appendPath(brakingConstVel)
	default:
		s.appendPath(ridingConstVel)
		s.appendPath(freewheeling)
	}
	if p.MinFreewheelPower <= power && power <= p.MaxFreewheelPower {
		s.distFreewheel += dist
	}
	if power < p.MinFreewheelPower {
		s.distBraking += dist
	}
}
