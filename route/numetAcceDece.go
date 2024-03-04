package route

import "github.com/pekkizen/motion"

func (s *segment) acceDeceVel(c *motion.BikeCalc, p par) {
	var (
		dist, time  float64
		jouleDrag   float64
		calcSteps   int
		vel         = s.vExit //***
		distLeft    = s.distLeft
		steps, Δvel = velSteps(s.vExit, s.vTarget, p.DeltaVel)
	)
	maxPedaled, riderPower := s.acceDecePower(p, Δvel > 0)

	for ; steps > 0; steps-- {

		power := riderPower
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, jDrag := c.DeltaVel(Δvel, vel, power)

		if Δtime <= 0 || Δtime > 1e4 {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		calcSteps++

		dist += Δdist
		if dist > distLeft {
			// interpolate values at road segment end
			I := 1 - (dist-distLeft)/Δdist
			Δvel *= I
			Δtime *= I
			Δdist *= I
			jDrag *= I
			dist = distLeft
			steps = -1
		}
		vel += Δvel
		time += Δtime
		jouleDrag -= jDrag
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
	if steps == 0 { // vTarget reached
		vel = s.vTarget
	}
	s.vExit = vel
	s.time += time
	s.jouleDrag += jouleDrag
	s.calcSteps += calcSteps
	s.distKinetic += dist
	if distLeft -= dist; distLeft < distTOL {
		distLeft = 0
	}
	s.distLeft = distLeft
	if Δvel < 0 {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheel)
	}
}

func (s *segment) acceDeceDist(c *motion.BikeCalc, p par) {
	var (
		dist, time   float64
		vel          = s.vExit
		steps, Δdist = s.distSteps(p.DeltaTime)
		accelerate   = s.vExit < s.vTarget
		decelerate   = !accelerate
	)
	maxPedaled, riderPower := s.acceDecePower(p, accelerate)

	for ; steps > 0; steps-- {

		power := riderPower
		if vel > maxPedaled {
			power = 0
		}
		Δvel, Δtime := c.DeltaDist(Δdist, vel, power)

		if Δtime < 0 || Δtime > 1e4 {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		vel += Δvel

		if accelerate && vel > s.vTarget || decelerate && vel < s.vTarget {
			I := 1 - (vel-s.vTarget)/Δvel
			Δvel *= I
			Δdist *= I
			Δtime *= I
			vel = s.vTarget
			steps = -1
		}
		time += Δtime
		dist += Δdist
		jDrag := -Δdist * c.Fdrag(vel-0.5*Δvel)
		s.jouleDrag += jDrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider += jDrag
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewh += jDrag
		}
	}
	s.vExit = vel
	s.time += time
	s.distKinetic += dist
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheel)
	}
}

func (s *segment) acceDeceTime(c *motion.BikeCalc, p par) {
	const deltaVelTOL = 1e-10 // m/s
	var (
		time, dist float64
		Δtime      = p.DeltaTime
		steps      = true
		vel        = s.vExit
		accelerate = s.vExit < s.vTarget
		decelerate = !accelerate
	)
	maxPedaled, riderPower := s.acceDecePower(p, accelerate)
	for steps {

		power := riderPower
		if vel > maxPedaled {
			power = 0
		}
		Δvel, Δdist := c.DeltaTime(Δtime, vel, power)

		if accelerate && Δvel < deltaVelTOL || decelerate && Δvel > -deltaVelTOL {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		dist += Δdist
		vel += Δvel

		if dist > s.distLeft {
			I := 1 - (dist-s.distLeft)/Δdist
			vel -= Δvel
			Δvel *= I
			vel += Δvel
			Δtime *= I
			Δdist *= I
			dist = s.distLeft
			steps = false
		}
		if accelerate && vel > s.vTarget || decelerate && vel < s.vTarget {
			I := 1 - (vel-s.vTarget)/Δvel
			Δvel *= I
			Δtime *= I
			dist -= Δdist
			Δdist *= I
			dist += Δdist
			vel = s.vTarget
			steps = false
		}
		time += Δtime
		jDrag := -Δdist * c.Fdrag(vel-0.5*Δvel)
		s.jouleDrag += jDrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider += jDrag
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewh += jDrag
		}
	}
	s.vExit = vel
	s.time += time
	s.distKinetic += dist
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
	if decelerate {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheel)
	}

}
