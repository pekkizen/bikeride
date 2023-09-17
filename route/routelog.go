package route

import (
	"bikeride/logerr"
	"math"
)

func (o *Route) Log(p par, l *logerr.Logerr) {

	if l.Mode() < 1 || l.Level() < 1 {
		return
	}
	o.checkRoad(p, l)
	o.checkRideSetup(p, l)
	o.checkRide(p, l)
}

func (o *Route) checkRide(p par, l *logerr.Logerr) {

	l.SetPrefix("ride:   ")
	const failLim = 30
	var (
		failed = 0
		r      = o.route
		round  = func(x float64) float64 { return math.Round(100*x) / 100 }
		pout   = p.PowerOut
	)
	for i := 1; i <= o.segments; i++ {
		s := &r[i]
		n := s.segnum

		if (s.calcPath == 4 || s.calcPath == 5) && s.vEntry != s.vExit {
			l.SegMsg(1, n, "Constant velocity and vEntry != vExit,", s.calcPath)
		}
		if s.vExit > p.Powermodel.MaxPedaledSpeed && s.powerRider > 0 && s.calcPath == 4 {
			l.SegMsg(1, n, "vExit =", round(s.vExit*3.6), " > maxPedaledSpeed & powerRider =",
				round(pout*s.powerRider), ",", s.calcPath)
		}
		if s.powerRider > 0 && s.powerBraking < 0 && s.calcPath != 34 && s.calcPath != 324 {
			l.SegMsg(3, n, "powerRider =", round(pout*s.powerRider), "& powerBraking =",
				round(s.powerBraking), ",", s.calcPath)
		}
		if s.powerBraking < 0 && s.vExit > s.vMax+0.5 {
			if failed++; failed < failLim {
				l.SegMsg(3, n, "Braking failed, powerBraking < 0 && vExit > vMax+0.5,", s.calcPath)
			}
		}
		if s.powerBraking < 0 && s.powerTarget > 1.1*p.Powermodel.FlatPower {
			if failed++; failed < failLim {
				l.SegMsg(1, n, "powerBraking < 0 & powerTarget > 1.1*flatPower,", s.calcPath)
			}
		}
		if p.Ride.MinSpeed > 0 {
			if s.vExit < p.Ride.MinSpeed*0.95 {
				l.SegMsg(1, n, "vExit < minSpeed,", s.calcPath)
			}
			if s.vExit == p.Ride.MinSpeed && s.powerRider > 1.25*p.Powermodel.UphillPower {
				l.SegMsg(1, n, "vExit = minSpeed and power > 1.25 x uphillPower,", s.calcPath)
			}
		}
	}
}

func (o *Route) checkRoad(p par, l *logerr.Logerr) {

	const (
		maxDIST  = 500.0
		minDist  = 0.5
		maxDELE  = 25.0
		maxGRADE = 0.24
		maxELE   = 10
	)
	l.SetPrefix("road:   ")
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		n := s.segnum
		if s.eleGPX < 0 {
			o.eleMissing++
			l.SegMsg(1, n, "Elevation < 0")
		}
		if math.Abs(s.grade*s.distHor) > maxDELE {
			l.SegMsg(3, n, "Delta elevation =", math.Round(100*s.grade*s.distHor)/100, "m")
		}
		if s.dist > maxDIST || s.dist < minDist {
			l.SegMsg(3, n, "Distance =", math.Round(100*s.dist)/100, "m")
		}
		if math.Abs(s.ele-s.eleGPX) > maxELE {
			l.SegMsg(3, n, "Elevation adjusted", math.Round(s.ele-s.eleGPX), "meters")
		}
		if math.Abs(s.grade) > maxGRADE {
			l.SegMsg(3, n, "Grade =", math.Round(100*s.grade), "%")
		}
	}
}

func (o *Route) checkRideSetup(p par, l *logerr.Logerr) {

	const veltol = 0.1 / 3.6
	l.SetPrefix("setup:  ")
	q := &p.Powermodel
	round := func(x float64) float64 { return math.Round(100*x) / 100 }
	if o.vMaxToNext > 0 {
		l.Msg(1, "Max speed set to next segment's max entry speed:", o.vMaxToNext, "segments")
	}
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		n := s.segnum
		// v := math.Round(1000*3.6*s.vTarget) / 1000
		v := 3.6 * s.vTarget

		if s.vTarget > s.vMax {
			l.SegMsg(1, n, "vTarget > vMax, ", s.calcPath)
		}
		minSpeed := p.Ride.MinSpeed
		if minSpeed > 0 {
			if s.vTarget < minSpeed {
				l.SegMsg(1, n, "vTarget < minSpeed")
			}
			if s.vTarget == minSpeed && s.powerTarget < q.UphillPower-4*powerTOL {
				l.SegMsg(1, n, "vTarget == minSpeed && powerTarget < uphillPower, ", s.calcPath)
			}
			if s.vTarget > minSpeed && s.powerTarget > q.UphillPower+4*powerTOL {
				l.SegMsg(1, n, "vTarget > minSpeed && powerTarget > uphillPower, ", s.calcPath)
			}
		}
		maxPedalled := q.MaxPedaledSpeed

		if s.vTarget > maxPedalled && s.powerTarget > 0 {
			l.SegMsg(1, n, "vTarget =", round(v), "> maxPedalledSpeed && powerTarget > 0, ", s.calcPath)
		}
		if s.vTarget < maxPedalled-veltol && s.vTarget < s.vMax && s.powerTarget <= 0 {
			l.SegMsg(1, n, "vTarget =", round(v),
				"< maxPedalledSpeed && vTarget < vMax && powerTarget =",
				round(s.powerTarget), ",", s.calcPath)
		}
		if s.powerTarget+2 < q.UphillPower && s.grade > q.UphillPowerGrade {
			l.SegMsg(1, n, "powerTarget < UphillPower && grade > uphillPowerGrade, ", s.calcPath)

		}
	}
}
