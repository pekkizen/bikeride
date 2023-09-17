package route

import (
	"math"
	"motion"
)

// Ride calculates the ride for the given parameters and route.
func (o *Route) Ride(c *motion.BikeCalc, p par) {
	var (
		prexit = 0.01 // must be > 0, start speed
		r      = o.route
		last   = o.segments
	)
	for i := 1; i <= last; i++ {
		s := &r[i]

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

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5 * c.MassKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)

	if s.jouleRider > 0 { // -> s.time - s.timeBraking > 0
		s.powerRider = s.jouleRider / (s.time - s.timeBraking)
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
		acceDecelerate(s, c, p) //decelerate with power >= 0

	case s.useConstantVel(c, p):

	default:
		acceDecelerate(s, c, p)
	}

	if s.distLeft > 0 {
		s.rideConstantVel(c, p)
	}
}

// here is some to think/do
func (s *segment) useConstantVel(c *motion.BikeCalc, p par) bool {
	const (
		maxSeconds  = 10
		velTolRel   = 0.05
		velTolMin   = 1e-5
		overUHpower = 1.025
	)
	dVel := math.Abs(s.vTarget - s.vEntry)

	if dVel > s.vEntry*velTolRel {
		return false
	}
	if dVel < velTolMin {
		return true
	}
	if !p.Ride.KeepEntrySpeed {
		return false
	}
	if s.dist > s.vEntry*maxSeconds {
		return false
	}
	power := c.PowerFromVel(s.vEntry)

	if power < 0 && s.vEntry < s.vTarget { // no unnecessary braking
		return false
	}
	if power > p.Powermodel.UphillPower*overUHpower {
		return false
	}
	if s.vEntry <= p.Powermodel.MaxPedaledSpeed && power >= 0 {
		return true
	}
	if s.vEntry > p.Powermodel.MaxPedaledSpeed && power <= 0 {
		return true
	}
	return false
}

func (s *segment) rideConstantVel(c *motion.BikeCalc, p par) {
	var (
		dist      = s.distLeft
		vExit     = s.vExit
		time      = dist / vExit
		jouleDrag = -dist * c.Fdrag(vExit)
	)
	s.jouleDrag += jouleDrag
	s.distLeft = 0
	s.time += time
	power := c.PowerFromVel(vExit)
	if math.Abs(power) < powerTOL {
		power = 0
	}
	joulePower := time * power
	switch {
	case power > 0:
		s.jouleRider += joulePower
		s.timeRider += time
		s.jouleDragRider += jouleDrag
		s.appendPath(ridingConstVel)

	case power < 0 && vExit > s.vMax-0.0001:
		s.jouleBraking += joulePower
		s.timeBraking += time
		s.distBraking += dist
		s.jouleDragBrake += jouleDrag
		s.powerBraking = s.jouleBraking / s.timeBraking
		s.appendPath(brakingConstVel)

	case power == 0: // rare case
		s.jouleDragFreewh += jouleDrag
		s.timeFreewheel += time
		s.distFreewheel += dist
		s.appendPath(freewheelConstV)

	default:
		// Should not be here. Braking for s.vExit < s.vMax.
		s.jouleSink += joulePower
		s.appendPath(7)
		s.appendPath(7)
		s.appendPath(7)
	}
}
