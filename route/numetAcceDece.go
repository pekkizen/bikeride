package route

import "github.com/pekkizen/motion"

func (s *segment) acceDeceVel(c *motion.BikeCalc, p par) {
	var (
		vel      = s.vExit //***
		distLeft = s.distLeft
	)
	var timeRider, jouleDrag, jouleRider, jouleDragRider,
		distRider, dist, time float64

	steps, Δvel := velSteps(vel, s.vTarget, p.OdeltaVel)
	maxPedaled, riderPower := s.acceDecePower(p, Δvel > 0)

	for ; steps > 0; steps-- {

		power := riderPower
		if vel > maxPedaled {
			power = 0
		}
		Δdist, Δtime, Δjdrag := c.DeltaVel(Δvel, vel, power)
		if Δtime <= 0 || Δtime > 60*60 {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++

		if dist += Δdist; dist >= distLeft {
			// linear interpolation of values at road segment end
			ipo := 1 - (dist-distLeft)/Δdist
			Δdist *= ipo // exact distance to end
			Δtime *= ipo
			Δvel *= ipo
			Δjdrag *= ipo
			dist = distLeft
			steps = -1
		}
		vel += Δvel
		time += Δtime
		jouleDrag -= Δjdrag
		if power > 0 {
			timeRider += Δtime
			jouleRider += Δtime * power
			jouleDragRider -= Δjdrag
			distRider += Δdist
		}
	}
	if steps == 0 { // vTarget reached, take exact vel
		vel = s.vTarget
	}
	s.vExit = vel
	s.time += time
	s.jouleDrag += jouleDrag
	s.distKinetic += dist
	s.distRider += distRider
	s.timeRider += timeRider
	s.jouleRider += jouleRider
	s.jouleDragRider += jouleDragRider

	if distLeft -= dist; distLeft < distTol {
		distLeft = 0
	}
	s.distLeft = distLeft
	if Δvel < 0 {
		s.jouleDeceRider = s.jouleRider
	}
	if timeRider == time {
		return
	}
	s.timeFreewheel += time - timeRider
	s.distFreewheel += dist - distRider
	s.jouleDragFreewh += jouleDrag - jouleDragRider
	if s.jouleRider == 0 {
		s.appendPath(freewheel)
	}
}

func (s *segment) acceDeceDist(c *motion.BikeCalc, p par) {
	var (
		dist, time   float64
		vel          = s.vExit
		steps, Δdist = s.distSteps(p.DeltaTime)
		acce         = s.vExit < s.vTarget
		dece         = !acce
	)
	maxPedaled, riderPower := s.acceDecePower(p, acce)

	for ; steps > 0; steps-- {
		power := riderPower
		if vel > maxPedaled {
			power = 0
		}
		Δvel, Δtime, Δjdrag := c.DeltaDist(Δdist, vel, power)

		if Δtime < 0 || Δtime > 1e5 {
			s.appendPath(noAcceleration)
			if dist == 0 {
				return
			}
			break
		}
		s.calcSteps++
		vel += Δvel
		if acce && vel >= s.vTarget || dece && vel <= s.vTarget {
			ipo := 1 - (vel-s.vTarget)/Δvel
			Δvel *= ipo
			Δjdrag *= ipo
			Δdist *= ipo
			Δtime *= ipo
			vel = s.vTarget
			steps = -1
		}
		time += Δtime
		dist += Δdist
		s.jouleDrag -= Δjdrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider -= Δjdrag
			s.distRider += Δdist
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewh -= Δjdrag
		}
	}
	s.vExit = vel
	s.time += time
	s.distKinetic += dist
	if s.distLeft -= dist; s.distLeft < distTol {
		s.distLeft = 0
	}
	if dece {
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
		acce       = s.vExit < s.vTarget
		dece       = !acce
	)
	maxPedaled, riderPower := s.acceDecePower(p, acce)
	for steps {
		power := riderPower
		if vel > maxPedaled {
			power = 0

		}
		Δvel, Δdist, Δjdrag := c.DeltaTime(Δtime, vel, power)

		if acce && Δvel < deltaVelTOL || dece && Δvel > -deltaVelTOL {
			s.appendPath(noAcceleration)
			if dist <= 0 {
				return
			}
			break
		}
		s.calcSteps++
		dist += Δdist
		vel += Δvel
		if dist >= s.distLeft {
			ipo := 1 - (dist-s.distLeft)/Δdist
			vel -= Δvel
			Δvel *= ipo
			vel += Δvel
			Δjdrag *= ipo
			Δtime *= ipo
			Δdist *= ipo
			dist = s.distLeft
			steps = false
		}
		if acce && vel >= s.vTarget || dece && vel <= s.vTarget {
			ipo := 1 - (vel-s.vTarget)/Δvel
			Δvel *= ipo
			Δjdrag *= ipo
			Δtime *= ipo
			dist -= Δdist
			Δdist *= ipo
			dist += Δdist
			vel = s.vTarget
			steps = false
		}
		time += Δtime
		s.jouleDrag -= Δjdrag
		if power > 0 {
			s.timeRider += Δtime
			s.jouleRider += Δtime * power
			s.jouleDragRider -= Δjdrag
			s.distRider += Δdist
		} else {
			s.timeFreewheel += Δtime
			s.distFreewheel += Δdist
			s.jouleDragFreewh -= Δjdrag
		}
	}
	s.vExit = vel
	s.time += time
	s.distKinetic += dist
	if s.distLeft -= dist; s.distLeft < distTol {
		s.distLeft = 0
	}
	if dece {
		s.jouleDeceRider = s.jouleRider
	}
	if s.jouleRider == 0 {
		s.appendPath(freewheel)
	}

}
