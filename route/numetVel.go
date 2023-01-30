package route

import "motion"


func (s *segment) acceDeceVel(c *motion.BikeCalc, p par, accelerate bool) {
	var (
		dist                = 0.0
		vel                 = s.vExit //***
		distLim             = s.distLeft
		decelerate          = !accelerate
		steps, Δvel         = s.velSteps(s.vTarget, p.DeltaVel)
		maxPedaled, powerAD = s.acceDeceVelPower(p, accelerate)
	)
	if p.DeltaVel >= 0.25 {
		distLim *= 0.95
	}
	for steps > 0 {
		steps--

		power := powerAD
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, fDrag, fSum := c.DeltaVel(Δvel, vel, power)

		if accelerate && fSum > -forceTOL || decelerate && fSum < forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++

		if dist += Δdist; dist > distLim {
			ipo := 1.0 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist = s.distLeft
			steps = -1
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
			s.jouleDragFreewh += jdrag
		}
	}
	s.vExit = vel
	if steps == 0 {
		s.vExit = s.vTarget
	}
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft < distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
	if s.timeFreewheel == s.time {
		s.appendPath(freewheeling)
	}
}
