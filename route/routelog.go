package route

import (
	"bikeride/logerr"
	"math"
)

// Log --
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
	r := o.route
	prev := &r[1]

	for i := 2; i <= o.segments; i++ {
		s := &r[i]
		n := s.segnum

		if (s.calcPath == 4 || s.calcPath == 5) && s.vEntry != s.vExit {
			l.SegMsg(1, n, "Constant velocity and vEntry != vExit,", s.calcPath)
		}
		if prev.vExit != s.vEntry {
			l.SegMsg(1, n, "vEntry != previous vExit,", s.calcPath)
		}
		if s.powerBraking < 0 && s.vExit > s.vMax+velTOL {
			l.SegMsg(1, n, "Braking failed, powerBraking < 0 && vExit > vMax,", s.calcPath)
		}
		if s.powerBraking < 0 && s.powerTarget > p.Powermodel.FlatPower {
			l.SegMsg(1, n, "powerBraking < 0 && powerTarget > flatPower,", s.calcPath)
		}
		if p.Ride.MinSpeed > 0 {
			if s.vExit < p.Ride.MinSpeed*0.95 {
				l.SegMsg(1, n, "vExit < minSpeed,", s.calcPath)
			}
			if s.vExit == p.Ride.MinSpeed && s.powerRider > 1.25*p.Powermodel.UphillPower {
				l.SegMsg(1, n, "vExit = minSpeed and power > 1.25 x uphillPower,", s.calcPath)
			}
		}
		prev = s

	}

}

func (o *Route) checkRoad(p par, l *logerr.Logerr) {

	const (
		maxDIST  = 500.0
		maxDELE  = 25.0
		maxGRADE = 0.2
	)
	l.SetPrefix("road:   ")
	maxFiltered := p.Filter.MaxFilteredEle
	maxIpo := p.Filter.MaxIpolations
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		n := s.segnum
		if s.eleGPX == 0 {
			o.eleMissing++
			l.SegMsg(0, n, "Elevation data missing")
		}
		if s.distHor < distTOL {
			l.SegMsg(1, n, "Distance < ", distTOL, "m")
		}
		if math.Abs(s.grade*s.distHor) > maxDELE {
			l.SegMsg(1, n, "Delta elevation >", maxDELE, "m")
		}
		if s.dist > maxDIST {
			l.SegMsg(1, n, "Distance >", maxDIST, "m")
		}
		if math.Abs(s.ele-s.eleGPX) > min(8, maxFiltered*0.5) && s.ipolations < maxIpo {
			l.SegMsg(1, n, "Elevation adjusted", math.Round(s.ele-s.eleGPX), "meters, interpolations", s.ipolations)
		}
		if math.Abs(s.grade) > maxGRADE {
			l.SegMsg(1, n, "Grade =", math.Round(100*s.grade), "%")
		}
		if s.ipolations == maxIpo {
			l.SegMsg(1, n, "Elevation adjusted", math.Round(s.ele-s.eleGPX), "meters, max interpolations", s.ipolations, "used")
		}
	}
}

func (o *Route) checkRideSetup(p par, l *logerr.Logerr) {

	const veltol = 0.1 / 3.6
	l.SetPrefix("setup:  ")
	P := &p.Powermodel
	minSpeed := p.Ride.MinSpeed
	if o.vMaxToNext > 0 {
		l.Msg(1, "Max speed set to next segment's max entry speed:", o.vMaxToNext, "segments")
	}
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		n := s.segnum
		v := math.Round(1000*3.6*s.vTarget) / 1000

		if s.vTarget > s.vMax {
			l.SegMsg(1, n, "vTarget > vMax", s.calcPath)
		}
		if minSpeed > 0 {
			if s.vTarget < minSpeed {
				l.SegMsg(1, n, "vTarget < minSpeed")
			}
			if s.vTarget == minSpeed && s.powerTarget < P.UphillPower-4*powerTOL {
				l.SegMsg(1, n, "vTarget == minSpeed && powerTarget < uphillPower", s.calcPath)
			}
			if s.vTarget > minSpeed && s.powerTarget > P.UphillPower+4*powerTOL {
				l.SegMsg(1, n, "vTarget > minSpeed && powerTarget > uphillPower", s.calcPath)
			}
		}
		maxPedalledSpeed := P.MaxPedaledSpeed
		if s.wind > 0 {
			maxPedalledSpeed -= s.wind
		}
		if s.vTarget > P.MaxPedaledSpeed && s.powerTarget > 0 {
			l.SegMsg(1, n, "vTarget (", v, ") > maxPedalledSpeed && powerTarget > 0 ", s.calcPath)
		}
		if s.vTarget < maxPedalledSpeed-veltol && s.vTarget < s.vMax && s.powerTarget <= 0 {
			l.SegMsg(1, n, "vTarget (", v, ") < maxPedalledSpeed && vTarget < vMax && powerTarget <= 0", s.calcPath)
		}
		if s.powerTarget+2 < P.UphillPower && s.grade > P.UphillPowerGrade {
			l.SegMsg(1, n, "powerTarget < UphillPower && grade > uphillPowerGrade")

		}
	}
}
