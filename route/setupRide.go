package route

import (
	"bikeride/motion"
	"bikeride/logerr"
	"math"
)

type ratioGenerator interface {
	Ratio(grade, wind float64) (ratio float64)
}

var errVelNotSolvable = errf(" Target velocity is not solvable")

//SetupRide ----
func (o *Route) SetupRide(c *motion.BikeCalc, power ratioGenerator, p par, l *logerr.Logerr) error {

	if p.UseVelTable {
		if e := setupTargetVelTable(c, power, p); e != nil {
			return e
		}
	}
	s := &o.route[o.segments+1]
	s.vEntryMax = p.MaxSpeed

	for i := o.segments; i > 0; i-- {
		next := s
		s = &o.route[i]
		if s.distHor <= distTol {
			continue
		}
		s.vMax = p.MaxSpeed
		c.SetGrade(s.grade)
		c.SetWind(s.wind)

		if e := s.targetVelAndPower(c, p, power); e != nil {
			return e
		}
		s.setMaxVel(o, c, p, next)	
		s.adjustTargetVelByMaxVel(c)
		if p.CheckRideSetup {
			s.checkRideSetup(c, p, l)
		}
	}
	return nil
}

func (s *segment) checkRideSetup(c *motion.BikeCalc, p par, l *logerr.Logerr) {

	P := p.Powermodel
	n := s.segnum
	if  math.Abs(s.powerTarget - c.PowerFromVel(s.vTarget)) > 2 {
		l.Msg(0, "seg",n, "abs(powerTarget - PowerFromVel(vTarget)) > 2")
	}
	if s.vTarget > s.vMax {
		l.Msg(0,"seg",n, "vTarget > vMax")
	}
	if s.vTarget < p.MinSpeed {
		l.Msg(0,"seg",n, "vTarget < minSpeed")
	}
	if s.vTarget == p.MinSpeed && s.powerTarget < P.UphillPower {
		l.Msg(0,"seg",n, "vTarget == minSpeed && powerTarget < uphillPower")
	}
	if s.vTarget > p.MinSpeed && s.powerTarget > P.UphillPower + 1 {
		l.Msg(0,"seg",n, "vTarget > minSpeed && powerTarget > uphillPower")
	}

	maxPedalledSpeed := P.MaxPedalledSpeed
	if s.wind > 0 {
		maxPedalledSpeed -= s.wind
	}
	if s.vTarget < maxPedalledSpeed && s.powerTarget == 0 {
		l.Msg(0,"seg",n, "vTarget < maxPedalledSpeed && powerTarget == 0")
	}
	if s.vTarget > P.MaxPedalledSpeed && s.powerTarget > 0 {
		l.Msg(0,"seg",n, "vTarget > maxPedalledSpeed && powerTarget > 0 ")
	}
	if  s.vTarget < maxPedalledSpeed && s.vTarget < s.vMax && s.powerTarget <= 0 {
		l.Msg(0,"seg",n, "vTarget < maxPedalledSpeed && vTarget < vMax && powerTarget <= 0")
	}

	if s.powerTarget + 2 < P.UphillPower && s.grade > P.UphillPowerGrade {
		l.Msg(0,"seg",n, "powerTarget < UphillPower && grade > uphillPowerGrade")
	}
}

func (s *segment) targetVelAndPower(c *motion.BikeCalc, p par, power ratioGenerator) error {

	P := p.Powermodel
	vel := 0.0
	ok := false
	if s.grade < p.MinPedalledGrade - 0.004*s.wind { 
		vel = c.VelFreewheeling() 
		if vel > P.MaxPedalledSpeed {
			s.vTarget = vel
			s.powerTarget = 0
			return nil
		}
	}
	if p.UseVelTable && s.grade > minTableGrade {
		vel, ok = s.velFromTable(c, p)
		if ok {
			return s.adjustTargetByMaxPedalled(c, p)
		}
	}
	s.powerTarget = P.FlatPower * power.Ratio(s.grade, s.wind)

	if s.powerTarget < p.MinPower {
		s.powerTarget = 0
	} 
	if s.vTarget, ok = c.VelFromPower(s.powerTarget, vel); !ok { // vel==0 -> use adjusted previous vel
		return errVelNotSolvable
	}
	return s.adjustTargetByMaxPedalled(c, p)
}

func (s *segment) adjustTargetByMaxPedalled(c *motion.BikeCalc, p par) (err error) {
	err = nil

	P := p.Powermodel
	if s.powerTarget == 0 {
		if s.vTarget > P.MaxPedalledSpeed {
			return 
		}
		if s.wind > P.SysHeadwind && s.grade < P.DownhillPowerGrade {
			return 
		}
	}
	if s.vTarget <= p.MinSpeed {
		s.vTarget = p.MinSpeed
		s.powerTarget = c.PowerFromVel(s.vTarget)
		return 
	}
	if s.powerTarget > 0 && s.vTarget <= P.MaxPedalledSpeed { 
		return 
	}

	//vTarget is on the wrong side of MaxPedalledSpeed
	s.vTarget = P.MaxPedalledSpeed
	s.powerTarget = c.PowerFromVel(s.vTarget)

	if s.powerTarget <= 0 {
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
		nextGradeLim 	= 0.0
		brakeDist 		= 7.5
		noLimitDist 	= 75.0
	)
	if p.LimitDownSpeeds && s.grade < p.VelLimitGrade && next.grade < nextGradeLim {
		if vDown := s.downhillMaxVel(c, p); s.vMax > vDown {
			s.vMax = vDown
		}
	}
	if p.LimitTurnSpeeds && s.radius > 0 {
		if vTurn := c.VelFromTurnRadius(s.radius, p.Ccf); s.vMax > vTurn {
			s.vMax = vTurn
		}
	}
	if p.LimitEntrySpeeds  {
		dist := max(brakeDist, s.dist/2)
		s.vEntryMax = c.MaxEntryVel(dist, s.vMax)
		if s.vMax > next.vEntryMax && s.dist < noLimitDist {
			s.vMax = next.vEntryMax
			o.vMaxToNext++
		}
	}
	if s.vMax < p.MinLimitedSpeed {
		s.vMax = p.MinLimitedSpeed
	}
}

func (s *segment) downhillMaxVel(c *motion.BikeCalc, p par) float64 {

	if p.BrakingDist > 0 {
		dist := p.BrakingDist	
		if s.grade < p.SteepDownhillGrade { // For steep downhills shorten braking distance
			dist *= p.SteepDownhillGrade / s.grade
		}
		return c.VelFromBrakeDist(dist)
		// return c.MaxEntryVel(dist, 0)
	}
	if p.VerticalDownSpeed > 0 {
		return (p.VerticalDownSpeed / -s.grade) * mh2ms
	}
	return p.MaxSpeed
}

func (s *segment) adjustTargetVelByMaxVel(c *motion.BikeCalc) {

	if s.vTarget < s.vMax - 0.05 { 
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

// velFromTable interpolates velocity from precalculated grade x velocity table.
// Not remarkable faster than eg. N-R, when setupTargetVelTable included.

const (
	gradeMax  		= 13
	gradeMin  		= -5
	minTableGrade	= 1.0*gradeMin / 100
	windMax   		= 5
	tableGlim 		= gradeMax - gradeMin 
	tableWlim 		= 2 * windMax
)

var velTable [tableGlim + 1][tableWlim + 1]float64

func (s *segment) velFromTable(c *motion.BikeCalc, p par) (float64, bool) {

	grade := s.grade*100 - gradeMin
	if grade < 0 {
		return 15, false
	}
	ok := true
	wind := s.wind + windMax
	if wind < 0 {
		wind = 0
		ok = false
	}
	g0 := int(grade)
	g1 := g0 + 1
	if g1  > tableGlim  {
		g0 = tableGlim
		ok = false
	}
	w0 := int(wind)
	w1 := w0 + 1
	if w1 > tableWlim {
		w0 = tableWlim
		ok = false
	}
	if !ok {
		return velTable[g0][w0], false
	}
	v00 := velTable[g0][w0]
	v01 := velTable[g0][w1]
	v10 := velTable[g1][w0]
	v11 := velTable[g1][w1]

	grade -= float64(g0) 
	wind  -= float64(w0)

	v0 := v00 + wind*(v01 - v00)
	v1 := v10 + wind*(v11 - v10)
	v :=  v0  + grade*(v1 - v0)

	s.vTarget = v
	s.powerTarget = c.PowerFromVel(s.vTarget)
	return v, true
}

func setupTargetVelTable(c *motion.BikeCalc, power ratioGenerator, p par) error {

	P := p.Powermodel
	prevel := 10.0
	for g := gradeMin; g <= gradeMax; g++ {
		grade := float64(g) * 0.01 
		c.SetGrade(grade)

		for w := -windMax; w <= windMax; w++ {
			var vel float64
			var ok bool
			wind := float64(w)
			c.SetWind(wind)
	
			pow := P.FlatPower * power.Ratio(grade, wind)

			if pow < p.MinPower {
				vel = c.VelFreewheeling()
				prevel = vel
				if vel >= P.MaxPedalledSpeed {
					velTable[g-gradeMin][w+windMax] = vel
					continue
				}
			}
			if pow >= p.MinPower {
				vel, ok = c.VelFromPower(pow, prevel-0.5)
				prevel = vel
				if !ok {
					return errVelNotSolvable
				}
				if vel <= P.MaxPedalledSpeed && vel >= p.MinSpeed {
					velTable[g-gradeMin][w+windMax] = vel
					continue
				}

			}
			if vel < p.MinSpeed {
				velTable[g-gradeMin][w+windMax] = p.MinSpeed
				continue
			}
			vel = P.MaxPedalledSpeed
			pow = c.PowerFromVel(vel)
			if pow < 0 {
				vel = c.VelFreewheeling()
			}
			velTable[g-gradeMin][w+windMax] = vel
		}
	}
	return nil
}
