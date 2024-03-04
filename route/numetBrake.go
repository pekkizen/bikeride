package route

import "github.com/pekkizen/motion"

func (s *segment) brake(c *motion.BikeCalc, p par, v0, v1 float64) {
	var (
		dist, time  float64
		jouleDrag   float64
		calcSteps   int
		vel         = v0
		distLeft    = s.distLeft
		steps, Δvel = velSteps(v0, v1, 2*p.DeltaVel)
	)
	if Δvel >= 0 {
		return
	}
	s.appendPath(braking)
	for ; steps > 0; steps-- {

		Δdist, Δtime, jDrag := c.DeltaVelBrake(Δvel, vel)

		if Δtime < 0 {
			s.appendPath(noAcceleration)
			return
		}
		calcSteps++
		dist += Δdist

		if dist > distLeft {
			I := 1 - (dist-distLeft)/Δdist
			Δvel *= I
			Δtime *= I
			jDrag *= I
			dist = distLeft
			steps = -1
		}
		time += Δtime
		jouleDrag -= jDrag
		vel += Δvel
	}
	if steps == 0 {
		vel = v1
	}
	s.vExit = vel
	s.calcSteps += calcSteps
	s.jouleBraking += -dist * c.Fbrake()
	s.jouleDrag += jouleDrag
	s.jouleDragBrake += jouleDrag
	s.time += time
	s.distBraking += dist
	s.timeBraking += time
	s.distKinetic += dist
	if distLeft -= dist; distLeft < distTOL {
		distLeft = 0
	}
	s.distLeft = distLeft
}

// decelerateDistance decelerates from v0 to v1 within the remaining distance
// s.distLeft. The deceleration force may be positive (braking) or negative
// (riding).
func (s *segment) decelerateDistance(c *motion.BikeCalc, p par, v0, v1 float64) {

	force, time, jForce, jDrag := c.DeltaVelDist(v0, v1, s.distLeft)

	if force > c.Fbrake() {
		s.brake(c, p, v0, v1)
		if s.distLeft > 0 {
			panic("decelerateDistance: distLeft > 0")
		}
		return
	}
	s.calcSteps++
	s.vExit = s.vExitMax
	s.time += time
	s.jouleDrag += -jDrag
	s.distKinetic += s.distLeft

	if force > 0 {
		s.jouleBraking += -jForce
		s.jouleDragBrake += -jDrag
		s.distBraking += s.distLeft
		s.timeBraking += time
		s.appendPath(slowDownBrake)
	} else {
		s.jouleRider += jForce
		s.jouleDragRider += -jDrag
		s.timeRider += time
		s.appendPath(slowDownRider)
	}
	s.distLeft = 0
}
