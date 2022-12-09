package route

import "bikeride/motion"
// import "math"

func (s *segment) accelerateT(c *motion.BikeCalc, p par) {

	maxPedalledSpeed := p.Powermodel.MaxPedalledSpeed
	poweracce := s.accelerationPower(p)
	Δtime := p.DeltaTime

	time, dist, power := 0.0, 0.0, 0.0
	steps := true
	v0 := s.vEntry
	for steps {
		s.calcSteps++
		power = poweracce
		if v0 > maxPedalledSpeed {
			power = 0
		}
		Δvel, Δdist := c.DeltaTime(Δtime, v0, power)
		if Δvel <= 0 {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		v := v0 + Δvel
		dist += Δdist
		if dist >= s.dist {
			s.appendPath(distExeeded)
			Δdistover := Δdist
			Δdist -= (dist - s.dist)
			ipo := Δdist / Δdistover
			Δvel *= ipo
			Δtime *= ipo
			v = v0 + Δvel
			dist = s.dist
			steps = false
		}
		if v >= s.vTarget {
			s.appendPath(vTargetExeeded)
			Δover := Δvel
			Δvel -= (v - s.vTarget)
			dist -= Δdist
			ipo := Δvel / Δover
			Δdist *= ipo
			Δtime *= ipo
			dist += Δdist
			v = s.vTarget
			steps = false
		}
		time += Δtime
		// s.jouleDrag -= Δdist * 0.5*(c.Fdrag(v0) + c.Fdrag(v0+Δvel))
		s.jouleDrag -= Δdist * c.Fdrag(v0+0.5*Δvel)
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

func (s *segment) decelerateT(c *motion.BikeCalc, p par) {

	powerdece := s.decelerationPower(p)
	Δtime := p.DeltaTime
	deceSpeedLim := s.vTarget + (p.Powermodel.MaxPedalledSpeed-s.vTarget)*p.SpeedDeceCoef

	time, dist, power := 0.0, 0.0, 0.0
	steps := true
	v0 := s.vExit
	for steps {
		s.calcSteps++
		power = 0
		if v0 < deceSpeedLim {
			power = powerdece
		}
		Δvel, Δdist := c.DeltaTime(Δtime, v0, power)
		if Δvel >= 0 {
			s.appendPath(wrongForce)
			if dist == 0 {
				return
			}
			break
		}
		v := v0 + Δvel
		dist += Δdist

		if dist >= s.distLeft {
			s.appendPath(distExeeded)
			Δdistover := Δdist
			Δdist -= (dist - s.distLeft)
			ipo := Δdist / Δdistover
			Δvel *= ipo
			Δtime *= ipo

			v = v0 + Δvel
			dist = s.distLeft
			steps = false
		}
		if v <= s.vTarget {
			s.appendPath(vTargetExeeded)
			Δover := Δvel
			Δvel += (s.vTarget - v)
			dist -= Δdist
			ipo := Δvel / Δover
			Δdist *= ipo
			Δtime *= ipo

			dist += Δdist
			v = s.vTarget
			steps = false
		}
		time += Δtime
		// s.jouleDrag -= Δdist * 0.5*(c.Fdrag(v0) + c.Fdrag(v0+Δvel))
		s.jouleDrag -= Δdist * c.Fdrag(v0+0.5*Δvel)
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
