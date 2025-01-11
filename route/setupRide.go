package route

import (
	"github.com/pekkizen/motion"
)

func (o *Route) SetupRide(c *motion.BikeCalc, power ratioGenerator, p par) error {
	setAccelerationStepping(p.AcceStepMode)
	c.SetMinPower(powerTol)

	var (
		s    = &o.route[o.segments+1]
		next *segment
	)
	s.vMax = 3.0
	for i := o.segments; i > 0; i-- {
		next, s = s, &o.route[i]

		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		if e := s.setTargetVelAndPower(c, p, power); e != nil {
			return e
		}
		s.adjustTargetVelByMaxMinPedaled(c, p)
		s.setMaxVel(c, p, next)
		s.calcJoulesAndTimeFromTargets(o)
	}
	return nil
}

func (s *segment) calcJoulesAndTimeFromTargets(o *Route) {
	timeTarget := s.dist / s.vTarget
	if s.powerTarget > 0 {
		o.JriderTarget += timeTarget * s.powerTarget
	}
	o.TimeTarget += timeTarget
}

func setAccelerationStepping(i int) {
	switch i {
	case stepVel: // 1
		acceDecelerate = (*segment).acceDeceVel
	case stepTime: // 2
		acceDecelerate = (*segment).acceDeceTime
	case stepDist: // 3
		acceDecelerate = (*segment).acceDeceDist
	default:
		acceDecelerate = (*segment).acceDeceVel
	}
}

func (s *segment) setTargetVelAndPower(c *motion.BikeCalc, p par, power ratioGenerator) (err error) {
	var (
		ok = false
		q  = &p.Powermodel
	)
	if s.grade < 0 && c.PowerFromVel(q.MaxPedaledSpeed) <= 0 {
		s.vFreewheel = c.VelFreewheel()
		s.vTarget = s.vFreewheel
		s.powerTarget = 0
		return nil
	}

	s.powerTarget = q.FlatPower * power.Ratio(s.grade, s.wind)
	s.vTarget, ok = c.VelFromPower(s.powerTarget, -1) // -1 -> use velguess function

	if !ok {
		return errNew(" setTargetVelAndPower: velocity is not solvable: " + c.Error())
	}
	return nil
}

func (s *segment) adjustTargetVelByMaxMinPedaled(c *motion.BikeCalc, p par) {
	var (
		maxPedaled = p.Powermodel.MaxPedaledSpeed
		minSpeed   = p.Ride.MinSpeed
	)
	if s.powerTarget == 0 && s.vTarget >= maxPedaled {
		return
	}
	if s.vTarget < minSpeed { // minspeed < 0 -> minspeed not set
		s.vTarget = minSpeed
		s.powerTarget = c.PowerFromVel(minSpeed)
		return
	}
	if s.powerTarget > 0 && s.vTarget <= maxPedaled {
		return
	}
	if s.powerTarget > 0 && s.vTarget > maxPedaled {
		s.vTarget = maxPedaled
		s.powerTarget = c.PowerFromVel(maxPedaled)
		return
	}
	if !test {
		return
	}
	if s.powerTarget == 0 && s.vTarget < maxPedaled {
		panic("adjustTargetVelByMaxMinPedaled: " +
			"powerTarget == 0 && vTarget < maxPedaled")
	}
	panic("adjustTargetVelByMaxMinPedaled: powerTarget < 0")
}

func (s *segment) setMaxVel(c *motion.BikeCalc, p par, next *segment) {
	const turnGradeLim = 1.0 / 100
	var (
		q    = &p.Ride
		vMax = q.MaxSpeed
	)
	if q.LimitDownSpeeds && s.grade < q.SpeedLimitGrade {
		if vDown := s.downhillMaxVel(c, p); vMax > vDown {
			vMax = vDown
		}
	}
	if q.LimitTurnSpeeds && s.radius > 0 && s.grade < turnGradeLim {
		if vTurn := c.VelFromTurnRadius(s.radius); vMax > vTurn {
			vMax = vTurn
		}
	}
	s.vExitMax = 9999
	if vMax > next.vMax {
		if q.LimitExitSpeeds {
			s.vExitMax = next.vMax
		}
		if vEntry := c.MaxEntryVel(s.dist, next.vMax); vMax > vEntry {
			vMax = vEntry
		}
	}
	if vMax < q.MinLimitedSpeed {
		vMax = q.MinLimitedSpeed
	}
	if s.vTarget < vMax && s.powerTarget < 0 { // no unnecessary braking
		s.powerTarget = 0
		s.vTarget = c.VelFreewheel()
	}
	if s.vTarget > vMax {
		s.vTarget = vMax
		s.powerTarget = c.PowerFromVel(vMax)
	}
	s.vMax = vMax
}

func (s *segment) downhillMaxVel(c *motion.BikeCalc, p par) (v float64) {
	q := &p.Ride
	v = q.MaxSpeed

	if q.BrakingDist > 0 {
		if v = c.MaxBrakeStopVel(q.BrakingDist); v > q.MaxSpeed {
			v = q.MaxSpeed
		}
		return
	}
	if q.VerticalDownSpeed > 0 {
		v = q.VerticalDownSpeed / -s.grade * mh2ms
		if s.grade < q.SteepDownhillGrade {
			v *= q.SteepDownhillGrade / s.grade
		}
		if v > q.MaxSpeed {
			v = q.MaxSpeed
		}
	}
	return
}
