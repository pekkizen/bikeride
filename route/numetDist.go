package route

import "bikeride/motion"

func (s *segment) distSteps(Δtime float64) (steps int, Δdist float64) {
	Δdist = s.vExit * Δtime
	if Δdist > s.distLeft {
		steps = 1
		Δdist = s.distLeft
		return
	}
	steps = int(s.distLeft/Δdist) + 1
	Δdist = s.distLeft / float64(steps)
	return
}

func (s *segment) accelerateD(c *motion.BikeCalc, p par) {

	maxPedalledSpeed := p.Powermodel.MaxPedalledSpeed
	poweracce := s.accelerationPower(p)
	steps, Δdist := s.distSteps(p.DeltaTime)

	time, dist, power := 0.0, 0.0, 0.0
	v0 := s.vEntry
	for steps > 0 {
		steps--
		s.calcSteps++

		power = 0
		if v0 < maxPedalledSpeed {
			power = poweracce
		}
		Δvel, Δtime := c.DeltaDist(Δdist, v0, power)

		if Δvel <= 0 {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		v := v0 + Δvel
		if v >= s.vTarget {
			s.appendPath(vTargetExeeded)
			ipo := (s.vTarget - v0) / Δvel
			Δdist *= ipo
			Δtime *= ipo
			v = s.vTarget
			steps = 0
		}
		dist += Δdist
		time += Δtime
		s.jouleDrag -= Δdist * c.Fdrag(0.5*(v0+v))
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
		}
		if power <= p.MaxFreewheelPower {
			s.distFreewheel += Δdist
		}
		v0 = v
	}
	s.vExit = v0
	s.time += time
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTol {
		s.distLeft = 0
	}
}

func (s *segment) decelerateD(c *motion.BikeCalc, p par) {

	powerdece := s.decelerationPower(p)
	steps, Δdist := s.distSteps(p.DeltaTime)
	deceSpeedLim := s.vTarget + (p.Powermodel.MaxPedalledSpeed-s.vTarget)*p.SpeedDeceCoef

	time, dist, power := 0.0, 0.0, 0.0
	v0 := s.vExit //!!!
	for steps > 0 {
		steps--
		s.calcSteps++

		power = 0
		if v0 < deceSpeedLim {
			power = powerdece
		}
		Δvel, Δtime := c.DeltaDist(Δdist, v0, power)

		if Δvel >= 0 {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		v := v0 + Δvel
		if v <= s.vTarget {
			s.appendPath(vTargetExeeded)
			ipo := (s.vTarget - v0) / Δvel
			Δdist *= ipo
			Δtime *= ipo
			v = s.vTarget
			steps = 0
		}
		time += Δtime
		dist += Δdist
		s.jouleDrag -= Δdist * c.Fdrag(0.5*(v0+v))
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
		}
		if power <= p.MaxFreewheelPower {
			s.distFreewheel += Δdist
		}
		v0 = v
	}
	s.jRiderDece = s.jouleRider
	s.vExit = v0
	s.time += time
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTol {
		s.distLeft = 0
	}
}
