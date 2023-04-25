package route

import "motion"

func (s *segment) brake(c *motion.BikeCalc, p par) {
	var (
		dist, time  float64
		jouleDrag   float64
		steps, Δvel = velSteps(s.vEntry, s.vMax, 2*p.DeltaVel)
		vel         = s.vEntry
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

		if dist += Δdist; dist > s.distLeft {
			ipo := 1 - (dist-s.distLeft)/Δdist
			Δvel *= ipo
			Δtime *= ipo
			Δdist *= ipo
			jDrag *= ipo
			dist = s.distLeft
			steps = -1
		}
		time += Δtime
		jouleDrag -= jDrag
		vel += Δvel
	}
	s.vExit = vel
	if steps == 0 {
		s.vExit = s.vMax
	}
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

// func (s *segment) brakeSingleStep(c *motion.BikeCalc, p par) {

// 	v0, v1 := s.vEntry, s.vMax

// 	dist, time, fDrag := c.BrakeSingle(v0, v1)

// 	if time <= 0 {
// 		s.appendPath(noAcceleration)
// 		return
// 	}
// 	s.calcSteps++
// 	if dist > s.distLeft {
// 		ipo := 1.0 - (dist-s.distLeft)/dist
// 		time *= ipo
// 		v1 = v0 + (v1-v0)*ipo
// 		dist = s.distLeft
// 	}
// 	s.vExit = v1
// 	s.time = time
// 	s.jouleBraking = -dist * c.Fbrake()
// 	s.distBraking = dist
// 	s.timeBraking = s.time
// 	s.powerBraking = s.jouleBraking / s.timeBraking
// 	s.distKinetic = dist
// 	s.jouleDrag = -dist * fDrag
// 	s.jouleDragBrake = s.jouleDrag
// 	if s.distLeft -= dist; s.distLeft < distTOL {
// 		s.distLeft = 0
// 	}
// }
