package route

import "bikeride/motion"

func (s *segment) acceDeceDist(c *motion.BikeCalc, p par, acce bool) {

	var (
		dist                float64
		v0                  = s.vExit
		steps, Δdist        = s.distSteps(p.DeltaTime)
		maxPedaled, powerAD = s.acceDeceVelPower(p, acce)
	)
	for steps > 0 {
		s.calcSteps++
		steps--

		power := powerAD
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δtime := c.DeltaDist(Δdist, v0, power)

		if acce && Δvel < 1e-6 || !acce && Δvel > -1e-6 {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
		v1 := v0 + Δvel
		if acce && v1 > s.vTarget || !acce && v1 < s.vTarget {
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
			s.jouleDragFreewheel += jdrag
		}
		v0 = v1
	}
	s.vExit = v0
	s.distKinetic += dist
	s.distLeft -= dist
	if s.distLeft <= distTOL {
		s.distLeft = 0
	}
	if !acce {
		s.jouleDeceRider = s.jouleRider
	}
}


func (s *segment) acceDeceTime(c *motion.BikeCalc, p par, acce bool) {

	var (
		time, dist          float64
		Δtime               = p.DeltaTime
		distLim             = s.distLeft
		steps               = true
		v0                  = s.vExit
		maxPedaled, powerAD = s.acceDeceVelPower(p, acce)
	)
	if Δtime > 0.25 {
		distLim *= 0.95
	}
	for steps {
		s.calcSteps++

		power := powerAD
		if v0 > maxPedaled {
			power = 0
		}
		Δvel, Δdist := c.DeltaTime(Δtime, v0, power)

		if acce && Δvel < 1e-6 || !acce && Δvel > -1e-6 {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break
		}
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
		} else if acce && v1 > s.vTarget || !acce && v1 < s.vTarget {
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
			s.jouleDragFreewheel += jdrag
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
	if !acce {
		s.jouleDeceRider = s.jouleRider
	}
}
