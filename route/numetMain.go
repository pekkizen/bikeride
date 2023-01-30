package route

import (
	"math"
	"motion"
)

func (s *segment) accelerate(c *motion.BikeCalc, p par) {
	s.appendPath(acceleration)

	switch p.DiffCalc {
	case 1:
		s.acceDeceVel(c, p, true)
	case 2:
		s.acceDeceDist(c, p, true)
	case 3:
		s.acceDeceTime(c, p, true)
	}
}

func (s *segment) decelerate(c *motion.BikeCalc, p par) {
	s.appendPath(deceleration)

	switch p.DiffCalc {
	case 1:
		s.acceDeceVel(c, p, false)
	case 2:
		s.acceDeceDist(c, p, false)
	case 3:
		s.acceDeceTime(c, p, false)
	}
}

func (s *segment) velSteps(vExit, Δvel float64) (steps int, ΔvelOut float64) {
	vrange := vExit - s.vExit
	steps = int(math.Abs(vrange)/Δvel) + 1
	ΔvelOut = vrange / float64(steps)
	return
}

func (s *segment) distSteps(Δtime float64) (steps int, Δdist float64) {
	Δdist = (s.vExit + s.vEntry) * 0.5 * Δtime
	steps = int(s.distLeft/Δdist) + 1
	Δdist = s.distLeft / float64(steps)
	return
}

func (s *segment) appendPath(i int) {
	s.calcPath *= 10
	s.calcPath += i
}

func (s *segment) acceDeceVelPower(p par, acceleration bool) (maxPedaled, power float64) {
	if acceleration {
		maxPedaled = p.Powermodel.MaxPedaledSpeed
		power = s.accelerationPower(p)
		return
	}
	maxPedaled = s.decelerationMaxPedalled(p)
	power = s.decelerationPower(p)
	return
}

func (s *segment) accelerationPower(p par) (power float64) {
	if s.powerTarget < -powerTOL {
		return 0
	}
	power = max(p.Ride.PowerAcce*s.powerTarget, p.Ride.PowerAcceMin)

	if power > p.Powermodel.UphillPower {
		power =  max(p.Powermodel.UphillPower, 1.05*s.powerTarget)
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

func (s *segment) decelerationMaxPedalled(p par) float64 {
	if s.vExit < p.Powermodel.FlatSpeed {
		return p.Powermodel.MaxPedaledSpeed
	}
	v := s.vTarget + (s.vExit-s.vTarget)*p.Ride.VelDeceLim
	return min(v, p.Powermodel.MaxPedaledSpeed)
}
