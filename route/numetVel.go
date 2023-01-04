package route

import "bikeride/motion"

func (s *segment) accelerateVel(c *motion.BikeCalc, p par) {

	var (
		maxPedaled  = p.Powermodel.MaxPedaledSpeed
		poweracce   = s.accelerationPower(p)
		steps, Δvel = velSteps(s.vEntry, s.vTarget, p.DeltaVel)
		vel         = s.vEntry
		distLim     = s.distLeft
		dist        = 0.0
	)
	if p.DeltaVel > 0.25 {
		distLim *= 0.95
	}
	for steps > 0 {
		s.calcSteps++

		power := poweracce
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if fSum > -forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		steps--
		dist += Δdist
		if dist > distLim {
			ipo := 1.0 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist = s.distLeft
			steps = 0
		}
		s.time += Δtime
		vel += Δvel
		jdrag := -Δdist * fDrag
		s.jouleDrag += jdrag
		if power > powerTOL {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider += jdrag
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewheel += jdrag
		}
	}
	s.vExit = vel
	s.distKinetic = dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
}

func (s *segment) decelerateVel(c *motion.BikeCalc, p par) {

	var (
		maxPedaled  = s.decelerationMaxPedalled(p)
		powerdece   = s.decelerationPower(p)
		steps, Δvel = velSteps(s.vExit, s.vTarget, p.DeltaVel)
		vel         = s.vExit // after possible braking
		distLim     = s.distLeft
		dist        = 0.0
	)
	if p.DeltaVel > 0.25 {
		distLim *= 0.95
	}
	for steps > 0 {
		s.calcSteps++

		power := powerdece
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if fSum < forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		steps--
		dist += Δdist
		if dist > distLim {
			ipo := 1.0 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist = s.distLeft
			steps = 0
		}
		s.time += Δtime
		vel += Δvel
		jdrag := -Δdist * fDrag
		s.jouleDrag += jdrag
		if power > powerTOL {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider += jdrag
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewheel += jdrag
		}
	}
	s.vExit = vel
	s.jouleDeceRider = s.jouleRider
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
}

func (s *segment) acceDeceVel(c *motion.BikeCalc, p par, acce bool) {

	var (
		dist                = 0.0
		vel                 = s.vExit
		distLim             = s.distLeft
		steps, Δvel         = velSteps(s.vEntry, s.vTarget, p.DeltaVel)
		maxPedaled, powerAD = s.acceDeceVelPower(p, acce)
	)
	if p.DeltaVel > 0.25 {
		distLim *= 0.95
	}
	for steps > 0 {
		s.calcSteps++
		steps--

		power := powerAD
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if acce && fSum > -forceTOL || !acce && fSum < forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		dist += Δdist
		if dist > distLim {
			ipo := 1.0 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist = s.distLeft
			steps = 0
		}
		s.time += Δtime
		vel += Δvel
		jdrag := -Δdist * fDrag
		s.jouleDrag += jdrag
		if power > powerTOL {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider += jdrag
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewheel += jdrag
		}
	}
	s.vExit = vel
	s.distKinetic = dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
	if !acce {
		s.jouleDeceRider = s.jouleRider
	}
	if s.timeFreewheel == s.time {
		s.appendPath(freewheeling)
	}
}
