package route

import "math"

// see velSteps below
var divBySteps = [21]float64{
	1.0, 1.0, 1.0 / 2, 1.0 / 3, 1.0 / 4, 1.0 / 5,
	1.0 / 6, 1.0 / 7, 1.0 / 8, 1.0 / 9, 1.0 / 10,
	1.0 / 11, 1.0 / 12, 1.0 / 13, 1.0 / 14, 1.0 / 15,
	1.0 / 16, 1.0 / 17, 1.0 / 18, 1.0 / 19, 1.0 / 20}

// velSteps calculates the number of steps and the step size for a
// velocity range v1-v0, so that steps * Δvel == v1 - v0.
func velSteps(v0, v1, oΔvIn float64) (steps int, Δvel float64) {
	dv := v1 - v0
	steps = int(math.Abs(dv)*oΔvIn) + 1
	if steps <= 20 {
		Δvel = dv * divBySteps[steps]
		return
	}
	Δvel = dv / float64(steps)
	return
}

func (s *segment) distSteps(Δtime float64) (steps int, Δdist float64) {

	Δdist = (s.vExit + s.vTarget) * 0.5 * Δtime
	steps = int(s.distLeft/Δdist) + 1
	Δdist = s.distLeft / float64(steps)
	return
}

// appendPath appends an operation code to the segment calculation path.
func (s *segment) appendPath(i int) {
	s.calcPath = 10*s.calcPath + i
}

func (s *segment) acceDecePower(p par, acce bool) (maxPedaled, power float64) {
	if acce {
		maxPedaled = p.Powermodel.MaxPedaledSpeed
		power = s.accelerationPower(p)
		s.appendPath(acceleration) // modify s.calcPath here!
		return
	}
	maxPedaled = s.decelerationMaxPedaled(p)
	power = s.decelerationPower(p)
	s.appendPath(deceleration)
	return
}

func (s *segment) accelerationPower(p par) (power float64) {
	q := &p.Ride
	if s.powerTarget < 0 {
		if s.powerTarget < -2*q.PowerAcceMin {
			return 0
		}
		return q.PowerAcceMin
	}
	power = q.PowerAcce * s.powerTarget
	if power < q.PowerAcceMin {
		return q.PowerAcceMin
	}
	if power > p.Powermodel.UphillPower {
		return max(p.Powermodel.UphillPower, 1.025*s.powerTarget)
	}
	return
}

func (s *segment) decelerationPower(p par) (power float64) {
	if s.powerTarget < powerTol {
		return 0
	}
	return p.Ride.PowerDece * s.powerTarget
}

func (s *segment) decelerationMaxPedaled(p par) (vel float64) {
	maxPedaled := p.Powermodel.MaxPedaledSpeed
	dVel := s.vExit - s.vTarget
	if dVel < 1 {
		return maxPedaled
	}
	vel = s.vTarget + dVel*p.Ride.VelDeceLim
	if vel > maxPedaled {
		return maxPedaled
	}
	return
}

/*
cosFromTanP22 returns inverse square root 1/math.Sqrt(1+tan^2) by a ratio of two
2. degree polynomials of tan^2. Max error ~ 6e-10 for abs(tan) < 0.3.
*/
// func cosFromTanP22(tan float64) (cos float64) {
// 	const (
// 		a2 = 0.73656502
// 		a4 = 0.05920391
// 		b2 = 1.2365650
// 		b4 = 0.3024874
// 	)
// 	tan *= tan
// 	cos = (1 + tan*(a2+tan*a4)) / (1 + tan*(b2+tan*b4))
// 	return
// }
