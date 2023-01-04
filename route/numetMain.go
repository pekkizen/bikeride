package route

import (
	"bikeride/motion"
	"math"
)

func velSteps(vEntry, vExit, Δvel float64) (steps int, ΔvelOut float64) {
	vrange := vExit - vEntry
	if math.Abs(vrange) < Δvel {
		steps = 1
		ΔvelOut = vrange
		return
	}
	steps = int(math.Abs(vrange)/Δvel) + 1
	ΔvelOut = vrange / float64(steps)
	return
}

func (s *segment) distSteps(Δtime float64) (steps int, Δdist float64) {
	Δdist = (s.vExit + s.vEntry) * 0.5 * Δtime
	if Δdist > s.distLeft {
		steps = 1
		Δdist = s.distLeft
		return
	}
	steps = int(s.distLeft/Δdist) + 1
	Δdist = s.distLeft / float64(steps)
	return
}

func (s *segment) appendPath(i int) {
	s.calcPath *= 10
	s.calcPath += i
}

func (s *segment) acceDeceVelPower(p par, acce bool) (maxPedaled, power float64) {
	if acce {
		maxPedaled = p.Powermodel.MaxPedaledSpeed
		power = s.accelerationPower(p)
	} else {
		maxPedaled = s.decelerationMaxPedalled(p)
		power = s.decelerationPower(p)
	}
	return
}

func (s *segment) accelerationPower(p par) float64 {
	P := &p.Powermodel

	if s.powerTarget < -powerTOL {
		return 0
	}
	power := max(p.Ride.PowerAcce*s.powerTarget, p.Ride.PowerAcceMin*P.FlatPower)

	if power > P.UphillPower {
		return max(P.UphillPower, 1.05*s.powerTarget)
	}
	return power
}

func (s *segment) decelerationPower(p par) float64 {
	if s.vExit > s.vMax {
		return 0
	}
	if s.powerTarget < powerTOL {
		return 0
	}
	return p.Ride.PowerDece * s.powerTarget
}

func (s *segment) decelerationMaxPedalled(p par) float64 {
	P := &p.Powermodel

	if s.vExit < P.FlatSpeed {
		return P.MaxPedaledSpeed
	}
	v := s.vTarget + (s.vExit-s.vTarget)*p.Ride.VelDeceLim
	return min(v, P.MaxPedaledSpeed)
}

func (s *segment) accelerate(c *motion.BikeCalc, p par) {
	s.appendPath(acceleration)
	switch p.DiffCalc {
	case 1:
		// s.accelerateVel(c, p)
		s.acceDeceVel(c, p, true)
	case 2:
		s.acceDeceDist(c, p, true)
	case 3:
		s.acceDeceTime(c, p, true)
	}

	if false { // *******************************
		s.accelerateVel(c, p)
		s.decelerateVel(c, p)
	}
}

func (s *segment) decelerate(c *motion.BikeCalc, p par) {
	s.appendPath(deceleration)
	switch p.DiffCalc {
	case 1:
		// s.decelerateVel(c, p)
		s.acceDeceVel(c, p, false)
	case 2:
		s.acceDeceDist(c, p, false)
	case 3:
		s.acceDeceTime(c, p, false)
	}
}
