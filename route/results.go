package route

import (
	"bikeride/motion"
	"math"
)

const (
	flatGRADE     = 1.0 / 100
	downhillGRADE = -4.0 / 100
	uphillGRADE   = 4.0 / 100
	dhSpeedGRADE  = -6.0 / 100
)

func (o *Route) Results(c *motion.BikeCalc, p par) *Results {

	r := &Results{VelMin: 99}
	r.calcRouteStats(o)
	o.calcRouteCourse()
	r.addRouteStats(o, p)
	r.addCalculatorStats(c)

	for i := 1; i <= o.segments; i++ {
		r.addSegment(&o.route[i], p)
	}
	r.calcMiscStats(c, p)
	r.energySums()
	r.riderEnergy(p)
	r.unitConversionOut()
	return r
}

func (r *Results) calcRouteStats(o *Route) {
	var (
		eleUp, eleDown, distot float64
		prevGrade, maxDist     float64
		maxGrade               = -99.0
		minGrade               = 99.0
		minDist                = 9999.0
		sumGradeNeg            = 0
	)
	for i := 1; i <= o.segments; i++ {

		s := &o.route[i]
		if s.distHor <= distTOL {
			continue
		}
		dist := s.dist
		grade := s.grade
		dEle := grade * s.distHor

		distot += dist
		if dEle < 0 {
			eleDown += dEle
		} else {
			eleUp += dEle
		}
		if grade < minGrade {
			minGrade = grade
		}
		if grade > maxGrade {
			maxGrade = grade
		}
		if dist < minDist {
			minDist = dist
		}
		if dist > maxDist {
			maxDist = dist
		}
		if prevGrade*grade < 0 {
			sumGradeNeg++
		}
		prevGrade = grade
	}
	o.distance = distot
	o.eleDown = -eleDown
	o.eleUp = eleUp
	o.distMean = distot / float64(o.segments)
	r.DistMean = o.distMean
	r.GradeSignChange = 100 * float64(sumGradeNeg) / float64(o.segments)
	r.DistMedian = o.distMedian
	// if o.segments < 50 {
	// 	r.DistMedian = o.distMean
	// }
	r.DistMin = minDist
	r.DistMax = maxDist
	r.MaxGrade = maxGrade
	r.MinGrade = minGrade
}

func (o *Route) calcRouteCourse() {
	s1 := &o.route[1]
	s2 := &o.route[o.segments+1]
	dLat := (s2.lat - s1.lat) * o.metersLat
	dLon := (s2.lon - s1.lon) * o.metersLon
	o.distDirect = math.Sqrt(dLon*dLon + dLat*dLat)
	if o.distDirect/o.distance < 0.25 {
		o.routeCourse = -1
		return
	}
	o.routeCourse = course(dLat/o.distDirect, dLon/o.distDirect) * rad2deg
}

func (r *Results) calcMiscStats(c *motion.BikeCalc, p par) {

	if r.CalcSegs > r.Segments/10 {
		r.CalcStepsAvg = float64(r.CalcSteps) / float64(r.CalcSegs)
		r.SingleStepPros /= float64(r.CalcSegs)
		r.SingleStepPros *= 100
	}
	r.VelAvg = r.DistTotal / r.Time
	if r.TimeDownhill > 0 {
		r.VelDownVert = r.VerticalDownEle / (r.TimeDownhill / 3600)
		r.VelDownhill = r.DistDownhill / r.TimeDownhill
	}
	c.SetWind(0)
	T := &p.Ride
	Q := &p.Powermodel

	if T.BrakingDist <= 0 {
		r.DownhillMaxSpeed = (T.VerticalDownSpeed / -dhSpeedGRADE) * mh2ms * ms2kmh
	} else {
		c.SetGrade(dhSpeedGRADE)
		r.DownhillMaxSpeed = c.VelFromBrakeDist(T.BrakingDist) * ms2kmh
	}
	if T.MinSpeed > 0 {
		r.MaxGradeUp = c.GradeFromVelAndPower(T.MinSpeed, Q.UphillPower) * 100
	}
	c.SetGrade(Q.DownhillPowerGrade)
	r.DownhillPowerSpeed, _ = c.NewtonRaphson(Q.DownhillPower/100*Q.FlatPower, 0.001, 6)
	r.DownhillPowerSpeed *= ms2kmh
	r.RhoBase, _ = c.RhoFromEle(p.BaseElevation)
}

func (r *Results) addCalculatorStats(c *motion.BikeCalc) {
	r.VelErrorMean = c.VelErrorMean()
	r.VelErrorAbsMean = c.VelErrorAbsMean()
	r.VelErrorPos = c.VelErrorPos()
	r.VelErrorMax = c.VelErrorMax()
	r.SolverRoundsAvg = float64(c.SolverRounds()) / max(1, float64(c.SolverCalls()))
	r.SolverCalls = c.SolverCalls()
	r.FreewheelCalls = c.FreewheelCalls()
	r.PowerFromVelCalls = c.PowerFromVelCalls()
	r.MaxIter = c.MaxIter()
}

func (r *Results) addRouteStats(o *Route, p par) {
	r.Segments = o.segments
	r.RouteCourse = o.routeCourse
	r.WindCourse = o.windCourse
	r.WindSpeed = o.windSpeed
	r.Temperature = o.Temperature
	r.BaseElevation = p.BaseElevation
	r.MeanElevation = o.EleMean
	r.AirPressure = p.AirPressure
	r.Rho = o.Rho
	r.TrkpErrors = o.trkpErrors
	r.TrkpRejected = o.trkpRejected
	r.DistTotal = o.distance
	r.DistDirect = o.distDirect
	r.EleUp = o.eleUp
	r.EleDown = o.eleDown
	r.EleUpGPX = o.eleUpGPX
	r.EleDownGPX = o.eleDownGPX
	r.EleMissing = o.eleMissing
	r.EleMean = o.EleMean
	r.EleLevelled = o.filter.eleLeveled
	// r.EleAveraged = o.filter.eleAveraged
	r.Ipolations = o.filter.ipolations
	r.Levelations = o.filter.levelations
	r.FilterRounds = o.filter.rounds
	r.JriderTotal = o.jouleRider
	r.Time = o.time
	r.LatMean = o.LatMean
	r.Gravity = o.Gravity
	r.Filtered = o.eleUpGPX - o.eleUp
	// r.Filterable = o.eleUpGPX - o.nonFilterable // take off?
	r.FilteredPros = 100 * r.Filtered / o.eleUpGPX
}

func (r *Results) addSegment(s *segment, p par) {

	if s.distHor <= distTOL {
		return
	}
	if s.jouleRider > 0 {
		r.addRider(s, p)
	}
	r.addJoules(s)
	r.addDists(s, p)
	if s.grade > 0 && s.jouleKinetic > 0 {
		r.addEleUpByMomentum(s)
	}
	if r.VelMax < s.vExit {
		r.VelMax = s.vExit
	}
	if r.VelMin > s.vExit {
		r.VelMin = s.vExit
	}
	r.TimeUHBreaks += s.timeBreak
	r.TimeTargetSpeeds += s.dist / s.vTarget
	r.TimeBraking += s.timeBraking

	r.DistFreewheel += s.distFreewheel
	r.TimeFreewheel += s.timeFreewheel

	if s.calcSteps > 0 {
		r.CalcSteps += s.calcSteps
		r.CalcSegs++
		if s.calcSteps == 1 {
			r.SingleStepPros++
		}
	}
}

func (r *Results) addEleUpByMomentum(s *segment) {

	Jforward := s.jouleKinetic + s.jouleDeceRider
	Jrelative := s.jouleKinetic / Jforward
	sin := s.grade / math.Sqrt(1+s.grade*s.grade)
	r.EleUpKinetic += Jrelative * sin * s.distKinetic
}

func (r *Results) addDists(s *segment, p par) {

	// if s.distFreewheel > 0 {
	// 	r.DistFreewheel += s.distFreewheel
	// 	r.TimeFreewheel += s.time
	// }

	r.DistBrake += s.distBraking

	grade := s.grade
	if math.Abs(grade) < flatGRADE {
		r.DistFlat += s.dist
	}
	if grade > uphillGRADE {
		r.DistUphill += s.dist
	}
	if grade < downhillGRADE {
		r.TimeDownhill += s.time
		r.DistDownhill += s.dist
		r.VerticalDownEle -= s.distHor * grade
	}

}

func (r *Results) addJoules(s *segment) {

	jouleDrag, jouleGrav, jouleKinetic := s.jouleDrag, s.jouleGrav, s.jouleKinetic

	jouleNetSum := s.jouleRider + s.jouleBraking + s.jouleRoll
	jouleNetSum += jouleDrag + jouleGrav + jouleKinetic
	r.SegEnergyMeanAbs += math.Abs(jouleNetSum)
	r.SegEnergyMean += jouleNetSum

	r.Jroll += s.jouleRoll
	r.Jbraking += s.jouleBraking

	if jouleDrag > 0 {
		r.JdragPush += jouleDrag
	} else {
		r.JdragRider += s.jouleDragRider
		r.JdragBrake += s.jouleDragBrake
		r.JdragFreewheel += s.jouleDragFreewheel
		r.JdragResistance += jouleDrag
	}

	if jouleGrav > 0 {
		r.JgravDown += jouleGrav
	} else {
		r.JgravUp += jouleGrav
	}
	if jouleKinetic > 0 {
		r.JkineticDece += jouleKinetic
	} else {
		r.JkineticAcce += jouleKinetic
	}
	if s.powerTarget > 0 {
		r.JfromTargetPower += s.powerTarget * s.dist / s.vTarget
	}
}

func (r *Results) addRider(s *segment, p par) {

	jouleDrag, jouleGrav := s.jouleDrag, s.jouleGrav
	jouleRider, jouleKinetic := s.jouleRider, s.jouleKinetic

	r.TimeRider += s.timeRider

	if s.powerRider >= p.UphillBreak.PowerLimit*p.Powermodel.UphillPower {
		r.TimeFullPower += s.timeRider
		r.JriderFullPower += jouleRider
	}
	if s.powerRider > p.Powermodel.FlatPower {
		r.TimeOverFlatPower += s.timeRider
	}
	Jforward := jouleRider
	if jouleKinetic > 0 {
		Jforward += jouleKinetic
	}
	if jouleGrav > 0 {
		Jforward += jouleGrav
	}
	if jouleDrag > 0 {
		Jforward += jouleDrag
	}
	JriderShare := -jouleRider / Jforward

	r.JriderRoll += JriderShare * s.jouleRoll
	if jouleGrav < 0 {
		r.JriderGravUp += JriderShare * jouleGrav
	}
	if jouleDrag < 0 {
		r.JriderDrag += JriderShare * jouleDrag
	}
	if jouleKinetic < 0 {
		r.JriderAcce += JriderShare * jouleKinetic
	}
}

func (r *Results) riderEnergy(p par) {
	if r.JriderTotal == 0 {
		return
	}
	r.JriderTotal *= p.PowerOut // !!!!!!!!!!!!
	r.PowerRiderAvg = r.JriderTotal / (r.Time - r.TimeBraking)

	r.BananaRider = r.JriderTotal * j2banana
	r.FatRider = r.JriderTotal * j2lard
	r.FoodRider = r.JriderTotal * j2kcal
	r.JlossDT = r.JriderTotal * p.DrivetrainLoss / 100

	// Kinetic energy by acceleration is not lost. It is later used
	// for resisting forces Fgrav, Froll and Fdrag.
	acceAdj := 1.0 + r.JriderAcce/(r.JriderGravUp+r.JriderDrag+r.JriderRoll)
	r.JriderGravUp *= acceAdj
	r.JriderDrag *= acceAdj
	r.JriderRoll *= acceAdj
}

func (r *Results) energySums() {

	r.SegEnergyMeanAbs /= float64(r.Segments)
	r.SegEnergyMean /= float64(r.Segments)

	j := -r.JriderAcce
	j -= r.JriderGravUp
	j -= r.JriderDrag
	j -= r.JriderRoll
	j -= r.JlossDT
	j += r.JriderTotal
	r.EnergySumRider = j

	j = r.JkineticDece
	j += r.JkineticAcce
	j += r.Jbraking
	j += r.JgravUp
	j += r.JgravDown
	j += r.JdragRider
	j += r.JdragBrake
	j += r.JdragFreewheel
	j += r.JdragPush
	j += r.Jroll
	j -= r.JlossDT
	j += r.JriderTotal
	r.EnergySumTotal = j
}

// unitConversionOut --
func (r *Results) unitConversionOut() {
	if r.WindCourse > 0 {
		r.WindCourse *= rad2deg
	}
	r.DistTotal *= m2km
	r.DistDirect *= m2km
	r.DistBrake *= m2km
	r.DistFreewheel *= m2km
	r.DistUphill *= m2km
	r.DistDownhill *= m2km
	r.DistFlat *= m2km
	r.MinGrade *= 100
	r.MaxGrade *= 100
	r.VelAvg *= ms2kmh
	r.VelMax *= ms2kmh
	r.VelMin *= ms2kmh
	r.VelDownhill *= ms2kmh
	r.Time *= s2h
	r.TimeTargetSpeeds *= s2h
	r.TimeUHBreaks *= s2h
	r.TimeFullPower *= s2h
	r.TimeOverFlatPower *= s2h
	r.TimeRider *= s2h
	r.TimeBraking *= s2h
	r.TimeFreewheel *= s2h
	r.TimeDownhill *= s2h

	//J* is from now on Wh*
	r.JriderTotal *= j2Wh
	r.JfromTargetPower *= j2Wh
	r.JriderFullPower *= j2Wh
	r.JriderGravUp *= j2Wh
	r.JriderDrag *= j2Wh
	r.JriderRoll *= j2Wh
	r.JriderAcce *= j2Wh
	r.JkineticDece *= j2Wh
	r.JkineticAcce *= j2Wh
	r.JdragRider *= j2Wh
	r.JdragFreewheel *= j2Wh
	r.JdragBrake *= j2Wh
	r.JdragPush *= j2Wh
	r.JdragResistance *= j2Wh
	r.Jroll *= j2Wh
	r.JlossDT *= j2Wh
	r.JgravUp *= j2Wh
	r.JgravDown *= j2Wh
	r.Jbraking *= j2Wh
	r.EnergySumRider *= j2Wh
	r.EnergySumTotal *= j2Wh
}
