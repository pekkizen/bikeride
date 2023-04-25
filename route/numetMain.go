package route

import (
	"math"
	"motion"
)

func (s *segment) acceDecelerate(c *motion.BikeCalc, p par) {
	if s.vExit < s.vTarget {
		s.appendPath(acceleration)
	} else {
		s.appendPath(deceleration)
	}
	switch p.DiffCalc {
	case 1:
		s.acceDeceVel(c, p)
	case 2:
		s.acceDeceDist(c, p)
	case 3:
		s.acceDeceTime(c, p)
	}
}

func  velSteps(v0, v1, Δvel float64) (steps int, ΔvelOut float64) {

	steps = int(math.Abs((v1-v0)/Δvel)) + 1
	ΔvelOut = (v1 - v0) / float64(steps)
	return
}

func (s *segment) distSteps(Δtime float64) (steps int, Δdist float64) {

	Δdist = min(10, (s.vExit + s.vTarget) * 0.5 * Δtime)
	// Δdist = s.vExit * Δtime
	steps = int(s.distLeft/Δdist) + 1
	Δdist = s.distLeft / float64(steps)
	return
}

func (s *segment) appendPath(i int) {
	s.calcPath *= 10
	s.calcPath += i
}

func (s *segment) acceDecePower(p par, acceleration bool) (maxPedaled, power float64) {
	if acceleration {
		maxPedaled = p.Powermodel.MaxPedaledSpeed
		power = s.accelerationPower(p)
		return
	}
	maxPedaled = s.decelerationMaxPedaled(p)
	power = s.decelerationPower(p)
	return
}

func (s *segment) accelerationPower(p par) (power float64) {
	if s.powerTarget < -p.Powermodel.FlatPower {
		return 0
	}
	power = max(p.Ride.PowerAcce*s.powerTarget, p.Ride.PowerAcceMin)

	if power > p.Powermodel.UphillPower {
		power = max(p.Powermodel.UphillPower, 1.05*s.powerTarget)
	}
	return
}

func (s *segment) decelerationPower(p par) (power float64) {
	if s.vExit > s.vMax {
		return 0
	}
	if s.powerTarget < powerTOL {
		return 0
	}
	return p.Ride.PowerDece * s.powerTarget
}

func (s *segment) decelerationMaxPedaled(p par) float64 {
	if s.vExit < p.Powermodel.FlatSpeed {
		return p.Powermodel.MaxPedaledSpeed
	}
	v := s.vTarget + (s.vExit-s.vTarget)*p.Ride.VelDeceLim
	return min(v, p.Powermodel.MaxPedaledSpeed)
}
