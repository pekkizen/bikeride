package route

import "motion"

func (s *segment) brake(c *motion.BikeCalc, p par) {
	var (
		dist, time  float64
		jouleDrag   float64
		vel         = s.vEntry
		steps, Δvel = velSteps(s.vEntry, s.vMax, 2*p.DeltaVel)
	)
	if Δvel >= 0 {
		return
	}
	s.appendPath(braking)
	for ; steps > 0; steps-- {

		Δdist, Δtime, jDrag := c.DeltaVelBrake(Δvel, vel)

		if Δtime <= 0 {
			s.appendPath(noAcceleration)
			return
		}
		s.calcSteps++
		dist += Δdist

		if dist > s.distLeft {
			I := 1 - (dist-s.distLeft)/Δdist
			Δvel *= I
			Δtime *= I
			Δdist *= I
			jDrag *= I
			dist = s.distLeft
			steps = -1
		}
		time += Δtime
		jouleDrag -= jDrag
		vel += Δvel
	}
	if steps == 0 {
		vel = s.vMax
	}
	s.vExit = vel
	s.jouleBraking = -dist * c.Fbrake()
	s.jouleDrag = jouleDrag
	s.jouleDragBrake = jouleDrag
	s.time = time
	s.distBraking = dist
	s.timeBraking = time
	s.powerBraking = s.jouleBraking / s.timeBraking
	s.distKinetic = dist
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
}
