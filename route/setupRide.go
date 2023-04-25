package route

import (
	"motion"
)

type ratioGenerator interface {
	Ratio(grade, wind float64) (ratio float64)
}

func (o *Route) SetupRide(c *motion.BikeCalc, power ratioGenerator, p par) error {

	c.SetMinPower(powerTOL) // tell Calculator to use the same min power tolerance
	if p.UseVelTable {
		if e := setupTargetVelTable(c, power, p); e != nil {
			return e
		}
	}
	r := o.route
	s := &r[o.segments+1]
	s.vEntryMax = p.Ride.MaxSpeed

	for i := o.segments; i > 0; i-- {
		next := s
		s = &r[i]

		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		if e := s.targetVelAndPower(c, p, power); e != nil {
			return e
		}
		s.setMaxVel(o, c, p, next)
		s.adjustTargetVelByMaxVel(c)
	}
	return nil
}

func (s *segment) targetVelAndPower(c *motion.BikeCalc, p par, power ratioGenerator) (err error) {
	vel := 0.0
	ok := false
	err = nil

	if s.grade < p.MinPedaledGrade-0.004*s.wind {
		if vel = c.VelFreewheeling(); vel > p.Powermodel.MaxPedaledSpeed {
			s.vTarget = vel
			s.powerTarget = 0
			return
		}
	}
	if p.UseVelTable && s.grade > minTableGrade {
		if vel, ok = s.velFromTable(c); ok {
			s.adjustTargetVelByMaxAndMinPedaled(c, p)
			return
		}
	}
	powerTarget := p.Powermodel.FlatPower * power.Ratio(s.grade, s.wind)

	if powerTarget < powerTOL {
		powerTarget = 0
	}
	if s.vTarget, ok = c.VelFromPower(powerTarget, vel); !ok { // vel==0 -> use adjusted previous vel
		return errf(" targetVelAndPower: velocity is not solvable")
	}
	s.powerTarget = powerTarget
	s.adjustTargetVelByMaxAndMinPedaled(c, p)
	return
}

func (s *segment) adjustTargetVelByMaxAndMinPedaled(c *motion.BikeCalc, p par) {

	maxPedaled := p.Powermodel.MaxPedaledSpeed
	minSpeed := p.Ride.MinSpeed

	if s.powerTarget == 0 && s.vTarget > maxPedaled {
		return
	}
	if s.vTarget < minSpeed {
		s.vTarget = minSpeed
		s.powerTarget = c.PowerFromVel(minSpeed)
		return
	}
	if s.powerTarget > 0 && s.vTarget <= maxPedaled {
		return
	}
	//vTarget is on the wrong side of MaxPedaledSpeed
	s.vTarget = maxPedaled
	s.powerTarget = c.PowerFromVel(maxPedaled)

	if s.powerTarget < powerTOL {
		s.powerTarget = 0
		s.vTarget = c.VelFreewheeling()
	}
	
}

func (s *segment) setMaxVel(o *Route, c *motion.BikeCalc, p par, next *segment) {
	const (
		turnGradeLim = 0.01
		nextGradeLim = 0
		minBrakeDist = 8
		noLimitDist  = 100
	)
	q := &p.Ride
	vMax := q.MaxSpeed

	if q.LimitDownSpeeds && s.grade < q.SpeedLimitGrade && next.grade <= nextGradeLim {
		if vDown := s.downhillMaxVel(c, p); vMax > vDown {
			vMax = vDown
		}
	}
	if q.LimitTurnSpeeds && s.radius > 0 && s.grade < turnGradeLim {
		if vTurn := c.VelFromTurnRadius(s.radius, q.Ccf); vMax > vTurn {
			vMax = vTurn
		}
	}
	if vMax < q.MinLimitedSpeed {
		vMax = q.MinLimitedSpeed
	}
	if q.LimitEntrySpeeds {
		brakeDist := max(minBrakeDist, s.dist*0.5)
		s.vEntryMax = c.MaxEntryVel(brakeDist, vMax)
		if vMax > next.vEntryMax && s.dist < noLimitDist {
			vMax = next.vEntryMax
			o.vMaxToNext++
		}
	}
	s.vMax = vMax
}

func (s *segment) downhillMaxVel(c *motion.BikeCalc, p par) float64 {
	q := &p.Ride

	if q.BrakingDist > 0 {
		dist := q.BrakingDist
		if s.grade < q.SteepDownhillGrade { // For steep downhills shorten braking distance?
			dist *= q.SteepDownhillGrade / s.grade
		}
		return c.MaxBrakeStopVel(dist)
	}
	if q.VerticalDownSpeed > 0 {
		return (q.VerticalDownSpeed / -s.grade) * mh2ms
	}
	return q.MaxSpeed
}

func (s *segment) adjustTargetVelByMaxVel(c *motion.BikeCalc) {

	if s.vTarget < s.vMax-0.05 {
		return
	}
	if s.vTarget < s.vMax {
		//increasing vTarget -> positive powers over MaxPedalledSpeed
		s.vMax = s.vTarget
		return
	}
	s.vTarget = s.vMax
	s.powerTarget = c.PowerFromVel(s.vMax)
}
