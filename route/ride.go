package route

import (
	"math"

	"github.com/pekkizen/motion"
)

const sameVelTol = 1e-4

// Holds the actual acce/deceleration function.
var acceDecelerate func(*segment, *motion.BikeCalc, par)

// Ride calculates the ride for the given parameters and route.
func (o *Route) Ride(c *motion.BikeCalc, p par) {
	prexit := 3.0 // must be > 0, start speed.
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]

		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		s.distLeft = s.dist
		s.vEntry = prexit
		s.vExit = prexit //***

		s.ride(c, p)

		prexit = s.vExit
		s.calcJoules(c)

		o.Time += s.time
		o.JouleRider += s.jouleRider
	}
}

func (s *segment) calcJoules(c *motion.BikeCalc) {

	s.jouleGrav = -s.dist * c.Fgrav()
	s.jouleRoll = -s.dist * c.Froll()
	s.jouleKinetic = 0.5 * c.WeightKin() * (s.vEntry*s.vEntry - s.vExit*s.vExit)

	// Powers are calculated over total segment time. Could also be
	// calculated over timeRider and timeBraking.
	s.powerRider = s.jouleRider / s.time
	s.powerBraking = s.jouleBrake / s.time
}

func (s *segment) ride(c *motion.BikeCalc, p par) {
	switch {
	case s.vExit > s.vMax:
		s.brake(c, p, s.vExit, s.vMax)
		if s.vExit == s.vTarget || s.distLeft == 0 {
			break
		}
		acceDecelerate(s, c, p)

	case s.useConstantVel(c, p):

	default:
		acceDecelerate(s, c, p)
	}

	switch {
	case s.distLeft == 0:

	case s.vExit <= s.vExitMax:
		s.rideConstantVel(c)

	// The following two are relevant only with limitExitSpeeds true, s.vExit > s.vExitMax
	case s.powerTarget > 0 && s.distLeft < 25:
		s.decelerateDistance(c, p, s.vExit, s.vExitMax)

	default:
		s.rideConstantVelAndBrakeAtEnd(c, p)
	}
}

// rideConstantVelAndBrakeAtEnd calculates constant speed ride and braking
// at the end of the segment. The calculation is done in reverse order,
// braking first.
func (s *segment) rideConstantVelAndBrakeAtEnd(c *motion.BikeCalc, p par) {
	initialVel := s.vExit
	s.brake(c, p, s.vExit, s.vExitMax)

	if s.distLeft > 0 {
		brakeExitVel := s.vExit
		s.vExit = initialVel
		s.calcPath /= 10
		s.rideConstantVel(c)
		s.appendPath(braking)
		s.vExit = brakeExitVel
	}
}

func (s *segment) useConstantVel(c *motion.BikeCalc, p par) bool {
	const (
		maxSeconds = 20
	)
	// if s.vFreewheel >= s.vMax {
	// 	return true
	// }
	if s.vExit > s.vExitMax { //!!!!!!!!!!
		return false
	}
	dVel := math.Abs(s.vTarget - s.vEntry)

	if dVel < sameVelTol {
		return true
	}
	if dVel > s.vEntry*p.Ride.KeepEntrySpeed {
		return false
	}
	if s.dist > s.vEntry*maxSeconds {
		return false
	}
	power := c.PowerFromVel(s.vEntry)

	if power < 0 && s.vEntry < s.vMax-sameVelTol { // no unnecessary braking
		return false
	}
	if power > p.Powermodel.UphillPower*(1+p.Ride.KeepEntrySpeed) {
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

func (s *segment) rideConstantVel(c *motion.BikeCalc) {
	var (
		dist      = s.distLeft
		vel       = s.vExit
		time      = dist / vel
		jouleDrag = -dist * c.Fdrag(vel)
		power     = c.PowerFromVel(vel)
	)
	if s.vExit == s.vFreewheel {
		power = 0
	}
	if math.Abs(power) < powerTol {
		power = 0
	}
	s.calcSteps++
	s.jouleDrag += jouleDrag
	s.distLeft = 0
	s.time += time
	joulePower := time * power

	switch {
	case power > 0:
		s.jouleRider += joulePower
		s.timeRider += time
		s.jouleDragRider += jouleDrag
		s.distRider += dist
		s.appendPath(ridingConstVel)

	case power < 0 && vel > s.vMax-1e-8:
		s.jouleBrake += joulePower
		s.timeBrake += time
		s.distBrake += dist
		s.jouleDragBrake += jouleDrag
		s.appendPath(brakingConstVel)

	case power == 0: // rare case
		s.timeFreewheel += time
		s.distFreewheel += dist
		s.jouleDragFreewh += jouleDrag
		s.appendPath(freewheelConstV)

	default: // braking under vMax
		// if test {
		// println("rideConstantVel",s.segnum, int(power+0.5), int(s.powerTarget+0.51),
		// int(3.6*s.vTarget+0.5), int(3.6*vel+0.5), int(10000*s.grade+0.5), s.calcPath)
		// }
		s.jouleSink += joulePower
		s.appendPath(5)
		s.appendPath(5)

	}
}
