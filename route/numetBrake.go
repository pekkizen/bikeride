package route

import "motion"

// brakeSteps brakes from s.vEntry to s.vMax
func (s *segment) brakeSteps(c *motion.BikeCalc, p par)  {
	s.appendPath(braking)
	var (
		dist, time  float64
		jouleDrag   float64
		steps, Δvel = s.velSteps(s.vMax, p.DeltaVel) 
		v0          = s.vEntry
	)
	for steps > 0 {
		steps--

		Δdist, fDrag, fSum := c.DeltaVelBrake(Δvel, v0)

		if fSum < forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return
			}
			break //fBrake + fDrag not enough anymore
		}
		s.calcSteps++
		
		if dist += Δdist; dist > s.distLeft {
			Δvel *= 1.0 - (dist-s.distLeft)/Δdist
			Δdist -= dist - s.distLeft
			dist = s.distLeft
			steps = -1
		}
		time += Δdist / (v0 + 0.5*Δvel)
		jouleDrag -= Δdist * fDrag
		v0 += Δvel
	}
	s.vExit = v0
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

// brake brakes from s.vEntry to s.vMax by single step
func (s *segment) brake(c *motion.BikeCalc, p par) {

	if !p.SingleStepBraking {
		 s.brakeSteps(c, p)
		 return
	}
	s.appendPath(braking)

	v0, v1 := s.vEntry, s.vMax

	dist, fDrag, fSum := c.DeltaVelBrake(v1-v0, v0)

	if fSum < forceTOL {
		s.appendPath(noForce)
		return
	}
	s.calcSteps += 1
	if dist > s.dist {
		v1 = v0 + (v1-v0)*s.dist/dist
		dist = s.dist
	}
	s.vExit = v1
	s.time = dist * 2 / (v0 + v1)
	s.jouleBraking = -dist * c.Fbrake()
	s.distBraking = dist
	s.timeBraking = s.time
	s.powerBraking = s.jouleBraking / s.timeBraking
	s.distKinetic = dist
	s.jouleDrag = -dist * fDrag
	s.jouleDragBrake = s.jouleDrag
	if s.distLeft -= dist; s.distLeft < distTOL {
		s.distLeft = 0
	}
}
