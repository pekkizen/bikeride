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
	o.checkRoad(l)
	o.checkRide(p, l)
}

func (o *Route) checkRide(p par, l *logerr.Logerr) {
	// constants defined in numetVel.go
	// acceleration   	= 1
	// braking        	= 2
	// deceleration   	= 3
	// ridingConstVel   	= 4
	// brakingConstVel  	= 5
	// freewheeling		= 0
	// distExeeded    	= 7
	// vTargetExeeded 	= 8
	// notEnoughForce 	= 9

	c1 := braking
	c2 := 100*braking + 10*wrongForce + brakingConstVel
	c3 := 10*deceleration + freewheeling
	c4 := 100*braking + 10*deceleration + freewheeling

	l.SetPrefix("setupRide:  ")
	if o.vMaxToNext > 0 {
		l.Msg(1, "Max speed set to next segment's max entry speed:", o.vMaxToNext, "segments")
	}
	l.SetPrefix("ride:       ")
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		if s.calcPath == c1 {
			l.FastMsg(2, s.segnum, "Braking to max speed failed")
		}
		if s.calcPath == c2 {
			l.FastMsg(1, s.segnum, "Brake force not enough")
		}
		if s.calcPath == c3 && s.vExit > s.vMax {
			l.FastMsg(2, s.segnum, "Freewheel "+"\""+"braking"+"\""+" to max speed failed")
		}
		if s.calcPath == c4 {
			l.FastMsg(3, s.segnum, "Braking + freewheeling")
		}
		if s.vExit == p.MinSpeed && s.powerRider > 1.25*p.Powermodel.UphillPower {
			l.FastMsg(1, s.segnum, "Speed = min speed and power > 1.25 x uphill power")
		}
	}

}

func (o *Route) checkRoad(l *logerr.Logerr) {
	l.SetPrefix("gpx:        ")

	if l.Mode() < 1 || l.Level() < 1 {
		return
	}
	for i := 1; i <= o.segments; i++ {
		s := &o.route[i]
		if s.eleGPX == 0 {
			o.eleMissing++
			l.FastMsg(0, s.segnum, "Elevation missing (=0)")
		}
		if s.distHor < minDist {
			if s.distHor == distTol {
				l.SegMsg(1, s.segnum, "Distance < ", distTol, "m")
			} else {
				l.SegMsg(1, s.segnum, "Distance <", minDist, "m")
			}
		}
		if math.Abs(s.grade*s.distHor) > maxDele {
			l.SegMsg(1, s.segnum, "Delta elevation >", maxDele, "m")
		}
		if s.dist > maxDist {
			l.SegMsg(1, s.segnum, "Distance >", maxDist, "m")
		}
	}
}
