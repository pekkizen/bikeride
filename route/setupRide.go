package route

import (
	// "bikeride/logerr"
	"bikeride/motion"
)

type ratioGenerator interface {
	Ratio(grade, wind float64) (ratio float64)
}

// SetupRide ----
func (o *Route) SetupRide(c *motion.BikeCalc, power ratioGenerator, p par) error {

	c.SetMinPower(powerTOL)
	if p.UseVelTable {
		if e := setupTargetVelTable(c, power, p); e != nil {
			return e
		}
	}
	s := &o.route[o.segments+1]
	s.vEntryMax = p.Ride.MaxSpeed

	for i := o.segments; i > 0; i-- {
		next := s
		s = &o.route[i]
		if s.distHor <= distTOL {
			continue
		}
		s.vMax = p.Ride.MaxSpeed
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

func (s *segment) targetVelAndPower(c *motion.BikeCalc, p par, power ratioGenerator) error {
	P := &p.Powermodel
	vel := 0.0
	ok := false

	if s.grade < p.MinPedaledGrade-0.004*s.wind {
		if vel = c.VelFreewheeling(); vel > P.MaxPedaledSpeed {
			s.vTarget = vel
			s.powerTarget = 0
			return nil
		}
	}
	if p.UseVelTable && s.grade > minTableGrade {
		if vel, ok = s.velFromTable(c); ok {
			return s.adjustTargetByMaxPedaled(c, p)
		}
	}
	s.powerTarget = P.FlatPower * power.Ratio(s.grade, s.wind)

	if s.powerTarget < powerTOL {
		s.powerTarget = 0
	}
	if s.vTarget, ok = c.VelFromPower(s.powerTarget, vel); !ok { // vel==0 -> use adjusted previous vel
		return errf(" Target velocity is not solvable")
	}
	return s.adjustTargetByMaxPedaled(c, p)
}

func (s *segment) adjustTargetByMaxPedaled(c *motion.BikeCalc, p par) (err error) {
	err = nil
	P := &p.Powermodel
	if s.powerTarget == 0 {
		if s.vTarget > P.MaxPedaledSpeed {
			return
		}
		if s.wind > P.SysHeadwind && s.grade < P.DownhillPowerGrade {
			return
		}
	}
	if s.vTarget < p.Ride.MinSpeed && p.Ride.MinSpeed > 0 {
		s.vTarget = p.Ride.MinSpeed
		s.powerTarget = c.PowerFromVel(s.vTarget)
		return
	}
	if s.powerTarget > 0 && s.vTarget <= P.MaxPedaledSpeed {
		return
	}
	//vTarget is on the wrong side of MaxPedalledSpeed
	s.vTarget = P.MaxPedaledSpeed
	s.powerTarget = c.PowerFromVel(s.vTarget)

	if s.powerTarget < powerTOL {
		s.powerTarget = 0
		s.vTarget = c.VelFreewheeling()
		return
	}
	if s.powerTarget > P.UphillPower {
		return errf(" Help! Can you free us from pedalling in this dark place!")
	}
	return
}

func (s *segment) setMaxVel(o *Route, c *motion.BikeCalc, p par, next *segment) {
	const (
		nextGradeLim = 0.0
		minBrakeDist    = 7.5
		noLimitDist  = 100.0
	)
	R := &p.Ride
	if R.LimitDownSpeeds && s.grade < R.VelLimitGrade && next.grade < nextGradeLim {
		if vDown := s.downhillMaxVel(c, p); s.vMax > vDown {
			s.vMax = vDown
		}
	}
	if R.LimitTurnSpeeds && s.radius > 0 {		
		if vTurn := c.VelFromTurnRadius(s.radius, R.Ccf); s.vMax > vTurn {
			s.vMax = vTurn
		}
	}
	if R.LimitEntrySpeeds {
		brakeDist := max(minBrakeDist, (s.dist*0.5))
		s.vEntryMax = c.MaxEntryVel(brakeDist, s.vMax)
		if s.vMax > next.vEntryMax && s.dist < noLimitDist {
			s.vMax = next.vEntryMax
			o.vMaxToNext++
		}
	}
	if s.vMax < R.MinLimitedSpeed {
		s.vMax = R.MinLimitedSpeed
	}
}

func (s *segment) downhillMaxVel(c *motion.BikeCalc, p par) float64 {
	R := &p.Ride

	if R.BrakingDist > 0 {
		dist := R.BrakingDist
		if s.grade < R.SteepDownhillGrade { // For steep downhills shorten braking distance
			dist *= R.SteepDownhillGrade / s.grade
		}
		return c.VelFromBrakeDist(dist)
	}
	if R.VerticalDownSpeed > 0 {
		return (R.VerticalDownSpeed / -s.grade) * mh2ms
	}
	return R.MaxSpeed
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
	s.powerTarget = c.PowerFromVel(s.vTarget)
}
