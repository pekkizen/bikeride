package route

import (
	"bikeride/motion"
	"math"
)

// func velStepsAcce(vEntry, vExit, Δvel float64) (steps int, ΔvelOut float64) {
// 	vrange := vExit - vEntry
// 	if vrange < Δvel {
// 		steps = 1
// 		ΔvelOut = vrange
// 		return
// 	}
// 	steps = int(vrange/Δvel) + 1
// 	ΔvelOut = vrange / float64(steps)
// 	return
// }

// func velStepsDece(vEntry, vExit, Δvel float64) (steps int, ΔvelOut float64) {
// 	vrange := vEntry - vExit
// 	if vrange < Δvel {
// 		steps = 1
// 		ΔvelOut = -vrange
// 		return
// 	}
// 	steps = int(vrange/Δvel) + 1
// 	ΔvelOut = -vrange / float64(steps)
// 	return
// }

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

func (s *segment) appendPath(i int) {
	s.calcPath *= 10
	s.calcPath += i
}

func (s *segment) accelerationPower(p par) float64 {
	P := &p.Powermodel

	if s.powerTarget < -P.FlatPower {
		return 0
	}
	powerAcceMin := p.PowerAcceMinPros * P.FlatPower
	if s.powerTarget <= 0 {
		return powerAcceMin
	}
	power := p.PowerAcceCoef * s.powerTarget

	if power > P.UphillPower {
		return P.UphillPower
	}
	if power < powerAcceMin {
		return powerAcceMin
	}
	return power
}

func (s *segment) decelerationPower(p par) float64 {
	if s.vExit > s.vMax {
		return 0
	}
	if s.powerTarget <= 0 {
		return 0
	}
	return p.PowerDeceCoef * s.powerTarget
}

func (s *segment) accelerate(c *motion.BikeCalc, p par) {
	s.appendPath(acceleration)

	if p.DiffCalc == 2 {
		s.accelerateD(c, p)
		return
	}
	if p.DiffCalc == 3 {
		s.accelerateT(c, p)
		return
	}
	maxPedalled := p.Powermodel.MaxPedalledSpeed
	poweracce := s.accelerationPower(p)
	steps, Δvel := velSteps(s.vEntry, s.vTarget, p.DeltaVel)
	dist := 0.0
	vel := s.vEntry

	for steps > 0 {
		s.calcSteps++

		power := poweracce
		if vel > maxPedalled {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if fSum > -minForce {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		steps--
		dist += Δdist
		if dist >= s.dist {
			Δover := dist - s.dist
			Δvel *= 1 - Δover/Δdist
			Δdist -= Δover
			Δtime = Δdist / (vel + 0.5*Δvel)
			dist = s.dist
			steps = -1
		}
		s.time += Δtime
		vel += Δvel

		s.jouleDrag -= Δdist * fDrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
		}
		if power < p.MaxFreewheelPower {
			s.distFreewheel += Δdist
		}
	}
	s.vExit = vel
	if steps == 0 {
		s.vExit = s.vTarget //exact, without rounding errors
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheeling)
	}
	s.distKinetic = dist
	s.distLeft = s.dist - dist
	if s.distLeft <= distTol {
		s.distLeft = 0
	}
}

func (s *segment) decelerate(c *motion.BikeCalc, p par) {
	s.appendPath(deceleration)

	if p.DiffCalc == 2 {
		s.decelerateD(c, p)
		return
	}
	if p.DiffCalc == 3 {
		s.decelerateT(c, p)
		return
	}
	powerdece := s.decelerationPower(p)
	steps, Δvel := velSteps(s.vExit, s.vTarget, p.DeltaVel)
	deceSpeedLim := s.vTarget + (p.Powermodel.MaxPedalledSpeed-s.vTarget)*p.SpeedDeceCoef

	dist := 0.0
	vel := s.vExit //sic, after possible braking

	for steps > 0 {
		s.calcSteps++

		power := powerdece
		if vel > deceSpeedLim {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if fSum < minForce {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		steps--
		dist += Δdist
		if dist >= s.distLeft {
			Δover := dist - s.distLeft
			Δvel *= 1 - Δover/Δdist
			Δdist -= Δover
			Δtime = Δdist / (vel + 0.5*Δvel)
			dist = s.distLeft
			steps = -1
		}
		s.time += Δtime
		vel += Δvel

		s.jouleDrag -= Δdist * fDrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
		}
		if power < p.MaxFreewheelPower {
			s.distFreewheel += Δdist
		}
	}
	s.vExit = vel
	if steps == 0 {
		s.vExit = s.vTarget
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheeling)
	}
	s.jRiderDece = s.jouleRider
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTol {
		s.distLeft = 0
	}
}

func (s *segment) brakeSteps(c *motion.BikeCalc, p par) bool {

	if s.powerTarget > p.Powermodel.FlatPower {
		return false
	}

	steps, Δvel := velSteps(s.vEntry, s.vMax, p.DeltaVel)
	dist := 0.0
	v := s.vEntry

	for steps > 0 {
		steps--
		s.calcSteps++

		Δdist, fDrag, fSum := c.DistBrake(v, v+Δvel)

		if fSum < minForce {
			s.appendPath(wrongForce)
			return true
		}
		dist += Δdist
		if dist >= s.dist {
			Δdistover := Δdist
			Δdist -= dist - s.dist
			Δvel *= Δdist / Δdistover
			dist = s.dist
			steps = -1
		}
		s.time += Δdist / (v + 0.5*Δvel)
		s.jouleDrag -= Δdist * fDrag
		v += Δvel
	}
	s.vExit = v
	if steps == 0 {
		s.vExit = s.vMax
	}
	s.jouleBraking = -dist * c.Fbrake()
	s.distBraking = dist
	s.timeBraking = s.time
	s.distKinetic = dist
	s.distLeft -= dist
	if s.distLeft < distTol {
		s.distLeft = 0
		return true
	}
	if s.powerTarget >= 0 && s.vExit > s.vTarget {
		return false
	}
	return true
}

// brake jarruttaa nopeudesta s.vEntry nopeuteen s.vMax >= s.vTarget
func (s *segment) brake(c *motion.BikeCalc, p par) bool {

	if s.powerTarget > p.Powermodel.FlatPower {
		return false
	}
	s.appendPath(braking)
	if !p.SingleStepBraking {
		return s.brakeSteps(c, p)
	}
	s.calcSteps = 1
	vExit := s.vMax
	vEntry := s.vEntry

	dist, fDrag, fSum := c.DistBrake(vEntry, vExit)

	if fSum < minForce {
		s.appendPath(wrongForce)
		return true
	}
	if dist > s.dist {
		vExit = vEntry + (vExit-vEntry)*s.dist/dist
		dist = s.dist
	}
	s.time = dist * 2 / (vEntry + vExit)
	s.timeBraking = s.time
	s.distBraking = dist
	s.vExit = vExit
	s.distKinetic = dist
	s.jouleDrag = -dist * fDrag
	s.jouleBraking = -dist * c.Fbrake()
	s.distLeft -= dist
	if s.distLeft < distTol {
		s.distLeft = 0
		return true
	}
	if s.powerTarget >= 0 && s.vExit > s.vTarget {
		return false
	}
	return true
}
