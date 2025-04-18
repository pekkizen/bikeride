package route

import "github.com/pekkizen/motion"

func (s *segment) brake(c *motion.BikeCalc, p par, v0, v1 float64) {
	var (
		dist, time  float64
		jouleDrag   float64
		vel         = v0
		distLeft    = s.distLeft
		steps, Δvel = velSteps(v0, v1, p.OdeltaVel)
	)
	if Δvel >= 0 {
		return
	}
	s.appendPath(braking)
	for ; steps > 0; steps-- {

		Δdist, Δtime, Δdrag := c.DeltaVelBrake(Δvel, vel)

		if Δtime < 0 {
			s.appendPath(noAcceleration)
			s.rideConstantVel(c) //??
			return
		}
		s.calcSteps++

		if dist += Δdist; dist > distLeft {
			ipo := 1 - (dist-distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdrag *= ipo
			dist = distLeft
			steps = -1
		}
		time += Δtime
		jouleDrag -= Δdrag
		vel += Δvel
	}
	if steps == 0 {
		vel = v1
	}
	s.vExit = vel
	s.jouleBrake += -dist * c.Fbrake()
	s.jouleDrag += jouleDrag
	s.jouleDragBrake += jouleDrag
	s.time += time
	s.distBrake += dist
	s.timeBrake += time
	s.distKinetic += dist
	if distLeft -= dist; distLeft < distTol {
		distLeft = 0
	}
	s.distLeft = distLeft
}
