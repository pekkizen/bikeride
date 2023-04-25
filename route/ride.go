package route

import (
	"math"
	"motion"
)

func (o *Route) Ride(c *motion.BikeCalc, p par) {

	prexit := max(3, p.Ride.MinSpeed)

	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]

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
		if s.powerTarget < 0 ||
			s.vExit <= s.vTarget ||
			s.distLeft == 0 {
			break
		}
		s.acceDecelerate(c, p) //decelerate

	case s.useConstantVel(c, p):

	default:
		s.acceDecelerate(c, p)
	}
	if s.distLeft > 0 {
		s.rideConstantVel(c, p)
	}
}

func (s *segment) useConstantVel(c *motion.BikeCalc, p par) bool {
	const (
		maxSeconds = 10
		velTolRel  = 0.075
		velTol     = 0.05 / 3.6
	)
	power := c.PowerFromVel(s.vEntry)
	if power <= 0 && s.vEntry < s.vTarget { // no unnecessary braking
		return false
	}
	dVel := math.Abs(s.vTarget - s.vEntry)
	if dVel < velTol {
		return true
	}
	P := &p.Powermodel
	if !p.Ride.KeepEntrySpeed ||
		dVel > s.vTarget*velTolRel ||
		s.dist > s.vEntry*maxSeconds ||
		(s.powerTarget >= P.UphillPower && s.vEntry > s.vTarget) {
		return false
	}
	if (s.vEntry < P.MaxPedaledSpeed && power > 0) ||
		(s.vEntry > P.MaxPedaledSpeed && power <= 0) {
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
	// s.calcSteps++
	s.jouleDrag += jouleDrag
	s.distLeft = 0
	s.time += time
	power := c.PowerFromVel(s.vExit)
	joulePower := time * power
	switch {
	case power > 0:
		s.jouleRider += joulePower
		s.timeRider += time
		s.jouleDragRider += jouleDrag
		s.appendPath(ridingConstVel)

	case power < 0 && s.vExit >= s.vMax:
		s.jouleBraking += joulePower
		s.timeBraking += time
		s.distBraking += dist
		s.jouleDragBrake += jouleDrag
		s.powerBraking = s.jouleBraking / s.timeBraking
		s.appendPath(brakingConstVel)

	case power > -1e-10:
		s.jouleDragFreewh += jouleDrag

	default:
		// Should not be here. Braking for s.vExit < s.vMax.
		s.jouleSink += joulePower
		// s.jouleBraking += joulePower
		s.appendPath(oddCase)
	}
}

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5 * c.MassKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)

	// if s.time > s.timeBraking {
	if s.jouleRider > 0 {
		s.powerRider = s.jouleRider / (s.time - s.timeBraking)
	}
}
