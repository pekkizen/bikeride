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

func (s *segment) acceDeceDist(c *motion.BikeCalc, p par, accelerate bool) {

	var (
		dist                float64
		v0                  = s.vExit
		decelerate          = !accelerate
		steps, Δdist        = s.distSteps(p.DeltaTime)
		maxPedaled, powerAD = s.acceDeceVelPower(p, accelerate)
	)
	for steps > 0 {
		steps--

		power := powerAD
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δtime := c.DeltaDist(Δdist, v0, power)

		if accelerate && Δvel < 1e-6 || decelerate && Δvel > -1e-6 {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		v1 := v0 + Δvel
		if accelerate && v1 > s.vTarget || decelerate && v1 < s.vTarget {
			ipo := 1.0 - (v1-s.vTarget)/Δvel
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			v1 = s.vTarget
			steps = 0
		}
		s.time += Δtime
		dist += Δdist
		jdrag := -Δdist * c.Fdrag(v0+0.5*Δvel)
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
		v0 = v1
	}
	s.vExit = v0
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
}

func (s *segment) acceDeceTime(c *motion.BikeCalc, p par, accelerate bool) {

	var (
		time, dist          float64
		Δtime               = p.DeltaTime
		distLim             = s.distLeft
		steps               = true
		v0                  = s.vExit
		decelerate          = !accelerate
		maxPedaled, powerAD = s.acceDeceVelPower(p, accelerate)
	)
	if Δtime > 0.25 {
		distLim *= 0.95
	}
	for steps {

		power := powerAD
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δdist := c.DeltaTime(Δtime, v0, power)

		if accelerate && Δvel < 1e-6 || decelerate && Δvel > -1e-6 {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		v1 := v0 + Δvel
		dist += Δdist

		if dist > distLim {
			ipo := 1.0 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			v1 = v0 + Δvel
			dist = s.distLeft
			steps = false
		} //else if	
		if accelerate && v1 > s.vTarget || decelerate && v1 < s.vTarget {
			dist -= Δdist
			ipo := 1.0 - (v1-s.vTarget)/Δvel
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist += Δdist
			v1 = s.vTarget
			steps = false
		}
		time += Δtime
		jdrag := -Δdist * c.Fdrag(v0+0.5*Δvel)
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
		v0 = v1
	}
	s.vExit = v0
	s.time += time
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
}
