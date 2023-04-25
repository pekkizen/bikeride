package route

import (
	"motion"
)

func (s *segment) acceDeceVel(c *motion.BikeCalc, p par) {
	var (
		Δdist, Δtime float64
		jDrag, dist  float64
		here         int
		vel          = s.vExit //***
		steps, Δvel  = velSteps(s.vExit, s.vTarget, p.DeltaVel)
	)
	maxPedaled, basePower := s.acceDecePower(p, Δvel > 0)

	for ; steps > 0; steps-- {
		
		power := basePower
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, jDrag = c.DeltaVel(Δvel, vel, power)

		if Δtime <= 0 || Δtime > 1e10 {
			if here++; here < 5 { // take off?
				Δvel *= 0.5
				steps *= 2
				steps++
				continue
			}
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++

		if dist += Δdist; dist > s.distLeft {
			d := 1 - (dist-s.distLeft)/Δdist
			Δvel *= d
			Δtime *= d
			Δdist *= d
			jDrag *= d
			dist = s.distLeft
			steps = -1
		}
		vel += Δvel
		s.time += Δtime
		s.jouleDrag -= jDrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider -= jDrag
		} else { 
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewh -= jDrag
		}
	}
	s.vExit = vel
	if steps == 0 {
		s.vExit = s.vTarget
	}
	s.distKinetic += dist
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
	if Δvel < 0 {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheeling)
	}
}

func (s *segment) acceDeceDist(c *motion.BikeCalc, p par) {
	var (
		dist         float64
		v0           = s.vExit
		steps, Δdist = s.distSteps(p.DeltaTime)
		accelerate   = s.vExit < s.vTarget
		decelerate   = !accelerate
	)
	maxPedaled, basePower := s.acceDecePower(p, accelerate)

	for ; steps > 0; steps-- {
		power := basePower
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δtime := c.DeltaDist(Δdist, v0, power)

		if accelerate && Δvel < deltaVelTOL || decelerate && Δvel > -deltaVelTOL {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		v1 := v0 + Δvel
		if accelerate && v1 > s.vTarget || decelerate && v1 < s.vTarget {
			d := 1 - (v1-s.vTarget)/Δvel
			Δvel *= d
			Δtime *= d
			Δdist *= d
			v1 = s.vTarget
			steps = -1
		}
		s.time += Δtime
		dist += Δdist
		jdrag := -Δdist * c.Fdrag(v0+0.5*Δvel)
		s.jouleDrag += jdrag
		if power > 0 {
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
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheeling)
	}

}

func (s *segment) acceDeceTime(c *motion.BikeCalc, p par) {
	var (
		time, dist float64
		Δtime      = p.DeltaTime
		steps      = true
		v0         = s.vExit
		accelerate = s.vExit < s.vTarget
		decelerate = !accelerate
	)
	maxPedaled, basePower := s.acceDecePower(p, accelerate)
	for steps {
		power := basePower
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δdist := c.DeltaTime(Δtime, v0, power)

		if accelerate && Δvel < deltaVelTOL || decelerate && Δvel > -deltaVelTOL {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		v1 := v0 + Δvel
		dist += Δdist

		if dist > s.distLeft {
			d := 1 - (dist-s.distLeft)/Δdist
			Δvel *= d
			Δtime *= d
			Δdist *= d
			v1 = v0 + Δvel
			dist = s.distLeft
			steps = false
		} // else if //wrong
		if accelerate && v1 > s.vTarget || decelerate && v1 < s.vTarget {
			dist -= Δdist
			d := 1 - (v1-s.vTarget)/Δvel
			Δvel *= d
			Δtime *= d
			Δdist *= d
			dist += Δdist
			v1 = s.vTarget
			steps = false
		}
		time += Δtime
		jdrag := -Δdist * c.Fdrag(v0+0.5*Δvel)
		s.jouleDrag += jdrag
		if power > 0 {
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
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheeling)
	}

}
