package route

import "bikeride/motion"

// brakeSteps brakes from s.vEntry to s.vMax > s.vTarget
func (s *segment) brakeSteps(c *motion.BikeCalc, p par) bool {

	s.appendPath(braking)
	steps, Δvel := velSteps(s.vEntry, s.vMax, p.DeltaVel)
	dist := 0.0
	v0 := s.vEntry

	for steps > 0 {
		steps--
		s.calcSteps++

		Δdist, fDrag, fSum := c.DistBrake(v0, v0+Δvel)

		if fSum < forceTOL {
			s.appendPath(noForce)
			if dist == 0 {
				return true
			}
			break
		}
		dist += Δdist
		if dist > s.distLeft {
			Δvel *= 1.0 - (dist-s.distLeft)/Δdist
			Δdist -= dist - s.distLeft
			dist = s.distLeft
			steps = -1
		}
		s.time += Δdist / (v0 + 0.5*Δvel)
		s.jouleDrag -= Δdist * fDrag
		v0 += Δvel
	}
	s.vExit = v0
	if steps == 0 {
		s.vExit = s.vMax
	}
	s.jouleBraking = -dist * c.Fbrake()
	s.distBraking = dist
	s.timeBraking = s.time
	s.powerBraking = s.jouleBraking / s.timeBraking
	s.distKinetic = dist
	s.jouleDragBrake = s.jouleDrag

	s.distLeft -= dist
	if s.distLeft < distTOL {
		s.distLeft = 0
		return true
	}
	if s.powerTarget > 0 && s.vExit > s.vTarget {
		return false
	}
	return true
}

// brake brakes from s.vEntry to s.vMax > s.vTarget by single step
func (s *segment) brake(c *motion.BikeCalc, p par) bool {

	if !p.SingleStepBraking && (s.vEntry-s.vMax)/p.DeltaVel > 2 {
		return s.brakeSteps(c, p)
	}
	s.appendPath(braking)
	s.calcSteps = 1
	vEntry, vExit := s.vEntry, s.vMax

	dist, fDrag, fSum := c.DistBrake(vEntry, vExit)

	if fSum < forceTOL {
		s.appendPath(noForce)
		return true
	}
	if dist > s.dist {
		vExit = vEntry - (vEntry-vExit)*s.dist/dist
		dist = s.dist
	}
	s.vExit = vExit
	s.time = dist * 2 / (vEntry + vExit)
	s.jouleBraking = -dist * c.Fbrake()
	s.distBraking = dist
	s.timeBraking = s.time
	s.powerBraking = s.jouleBraking / s.timeBraking
	s.distKinetic = dist
	s.jouleDrag = -dist * fDrag
	s.jouleDragBrake = s.jouleDrag

	s.distLeft -= dist
	if s.distLeft < distTOL {
		s.distLeft = 0
		return true
	}
	if s.powerTarget > 0 && s.vExit > s.vTarget {
		return false
	}
	return true
}
