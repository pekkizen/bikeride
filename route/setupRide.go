package route

import (
	"github.com/pekkizen/motion"
)

func (o *Route) SetupRide(c *motion.BikeCalc, power ratioGenerator, p par) error {

	setAccelerationStepping(p.IntegralType) // select deltaVel, deltaTime or deltaDist stepping
	c.SetMinPower(powerTOL)                 // tell Calculator to use the same min power tolerance

	if p.UseVelTable {
		if !fillTargetVelTable(c, power, p.Powermodel.FlatPower) {
			return errNew(" SetupRide: fillTargetVelTable failed\n" + c.Error())
		}
	}
	var (
		s    = &o.route[o.segments+1]
		next *segment
	)
	s.vMax = 1.0

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
		ok  bool
		vel = -1.0 //vel < 0 -> VelFromPower uses internal velguess function.
	)
	if s.grade < -p.Bike.Crr { // if c.Fgr() < 0
		s.vFreewheel = c.VelFreewheeling()
		if s.vFreewheel > p.Powermodel.MaxPedaledSpeed {
			s.vTarget = s.vFreewheel
			s.powerTarget = 0
			return nil
		}
	}
	if p.UseVelTable {
		if vel, ok = velFromTable(s.grade, s.wind); ok {
			s.vTarget = vel
			s.powerTarget = c.PowerFromVel(vel)
			// Uncomment lines below for velTable velocities error calculation.
			// p.VelErrors = true
			// powerTarget := p.Powermodel.FlatPower * power.Ratio(s.grade, s.wind)
			// c.VelError(powerTarget, vel)
			return nil
		}
		// vel goes to velguess for VelFromPower
	}
	////////////////////////////////////////////////////////////////////////
	s.powerTarget = p.Powermodel.FlatPower * power.Ratio(s.grade, s.wind)
	s.vTarget, ok = c.VelFromPower(s.powerTarget, vel)
	////////////////////////////////////////////////////////////////////////

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
	if TEST && !(s.powerTarget > 0 && s.vTarget > maxPedaled) {
		panic("adjustTargetVelByMaxMinPedaled: should not be here")
	}
	s.vTarget = maxPedaled
	s.powerTarget = c.PowerFromVel(maxPedaled)

	if s.powerTarget >= 0 {
		return
	}
	s.powerTarget = 0
	s.vFreewheel = c.VelFreewheeling()
	s.vTarget = s.vFreewheel
	if TEST {
		panic("adjustTargetVelByMaxMinPedaled: s.powerTarget < 0 ")
	}
}

func (s *segment) setMaxVel(c *motion.BikeCalc, p par, next *segment) {
	const (
		turnGradeLim = 1.0 / 100
	)
	q := &p.Ride
	vMax := q.MaxSpeed

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
	if vMax < q.MinLimitedSpeed {
		vMax = q.MinLimitedSpeed
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
	if s.vTarget > vMax {
		s.vTarget = vMax
		s.powerTarget = c.PowerFromVel(vMax)
	}
	s.vMax = vMax
}

func (s *segment) downhillMaxVel(c *motion.BikeCalc, p par) (speed float64) {
	q := &p.Ride
	speed = q.MaxSpeed

	if q.BrakingDist > 0 {
		speed = c.MaxBrakeStopVel(q.BrakingDist)
		// if s.grade < q.SteepDownhillGrade {
		// 	speed *= q.SteepDownhillGrade / s.grade
		// }
		return
	}
	if q.VerticalDownSpeed > 0 {
		speed = q.VerticalDownSpeed / -s.grade * mh2ms
		if s.grade < q.SteepDownhillGrade {
			speed *= q.SteepDownhillGrade / s.grade
		}
		return
	}
	return
}
