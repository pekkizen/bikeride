package route

import (
	"math"

	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/motion"
)

const (
	flatGRADE     = 1.0 / 100
	downhillGRADE = -4.0 / 100
	uphillGRADE   = 4.0 / 100
	dhSpeedGRADE  = -6.0 / 100
)

// CheckSum implements a simple checksum to detect change in results.
func (r *Results) CheckSum() uint64 {
	u := math.Float64bits(r.Time) << 44 >> 44
	u += math.Float64bits(r.DistTotal) << 44 >> 24
	u += math.Float64bits(r.EnergySumTotal) << 40
	return splitmix(u)
}
func splitmix(x uint64) uint64 {
	z := x + 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (o *Route) Results(c *motion.BikeCalc, p par, l *logerr.Logerr) *Results {

	r := &Results{VelMin: 99}
	r.calcRouteStats(o)
	o.calcRouteCourse()
	r.addRouteStats(o, p)
	r.addCalculatorStats(c)

	for i := 1; i <= o.segments; i++ {
		r.addRoadSegment(&o.route[i], p)
	}
	r.calcMiscStats(c, p)
	r.energySums()
	r.riderEnergy(p)
	r.unitConversionOut()
	return r
}

func (r *Results) checkEnergySums(l *logerr.Logerr) {
	e1 := r.SegEnergyMean * float64(r.Segments)
	e2 := r.EnergySumTotal / j2Wh
	if math.Abs(e1-e2) > 1e-5 {
		l.Msg(0, "We have a problem!\n",
			"Energy net sums difference > 1e-5\n",
			"Segment energy net sum (J)", e1, "\n",
			"Total energy net sum (J)  ", e2, "\n")
	}
}

func (r *Results) calcRouteStats(o *Route) {
	var (
		eleUp, eleDown     float64
		prevGrade, maxDist float64
		distsum            float64
		sumGradeChange     float64
		prevDist           = o.distMedian
		maxGrade           = -999.0
		minGrade           = 999.0
		minDist            = 9999.0
		maxEle             = -9999.0
		minEle             = 9999.0
		last               = o.segments
	)

	for i := 1; i <= last; i++ {
		var (
			s     = &o.route[i]
			ele   = s.ele
			dist  = s.dist
			grade = s.grade
			dEle  = s.grade * s.distHor
		)
		distsum += dist
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
		if ele < minEle {
			minEle = ele
		}
		if ele > maxEle {
			maxEle = ele
		}
		// grade change / 1 meter
		sumGradeChange += math.Abs(grade-prevGrade) * 2 / (dist + prevDist)
		prevGrade = grade
		prevDist = dist
	}
	o.distance = distsum
	o.eleDown = -eleDown
	o.eleUp = eleUp
	o.eleMin = minEle
	o.eleMax = maxEle
	o.distMean = distsum / float64(o.segments)
	r.DistMean = o.distMean
	// mean grade% change per 10 meters of distance
	r.RelGradeChange = (10 * 100) * sumGradeChange / float64(o.segments)
	r.DistMedian = o.distMedian
	r.DistMin = minDist
	r.DistMax = maxDist
	r.MaxGrade = maxGrade
	r.MinGrade = minGrade
	// dDdistTot := o.distGPX - o.distLine
	// dDistFilt := o.distGPX - o.distance
	// r.FilteredDistPros = 100 * dDistFilt / dDdistTot
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
	o.routeCourse = course(dLon, dLat) * (180 / π)
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
	t := &p.Ride
	q := &p.Powermodel

	if t.BrakingDist <= 0 {
		r.DownhillMaxSpeed = (t.VerticalDownSpeed / -dhSpeedGRADE) * mh2ms * ms2kmh
	} else {
		c.SetGradeExact(dhSpeedGRADE)
		r.DownhillMaxSpeed = c.MaxBrakeStopVel(t.BrakingDist) * ms2kmh
	}
	if t.MinSpeed > 0 {
		r.MaxGradeUp = c.GradeFromVelAndPower(t.MinSpeed, q.UphillPower) * 100
	}
	c.SetGradeExact(q.DownhillPowerGrade)
	r.DownhillPowerSpeed, _ = c.NewtonRaphson(q.DownhillPower/100*q.FlatPower, 0, 8)
	r.DownhillPowerSpeed *= ms2kmh
	r.RhoBase, _ = c.RhoFromEle(p.Environment.BaseElevation)
}

func (r *Results) addCalculatorStats(c *motion.BikeCalc) {
	r.VelErrorMean = c.VelErrorMean()
	r.VelErrorAbsMean = c.VelErrorAbsMean()
	r.VelErrorPos = c.VelErrorPos()
	r.VelErrorMax = c.VelErrorMax()
	r.SolverRoundsAvg = float64(c.SolverRounds()) / max(1, float64(c.SolverCalls()))
	r.SolverCalls = c.SolverCalls()
	r.MaxIter = c.MaxIter()
}

func (r *Results) addRouteStats(o *Route, p par) {
	r.Segments = o.segments
	r.RouteCourse = o.routeCourse
	r.WindCourse = o.windCourse
	r.WindSpeed = o.windSpeed
	r.Temperature = o.Temperature
	r.BaseElevation = p.Environment.BaseElevation
	r.MeanElevation = o.EleMean
	r.AirPressure = p.Environment.AirPressure
	r.Rho = o.Rho
	r.TrkpErrors = o.trkpErrors
	r.TrkpRejected = o.trkpRejected
	r.DistTotal = o.distance
	r.DistLine = o.distLine
	r.DistGPX = o.distGPX
	r.DistDirect = o.distDirect
	r.EleUp = o.eleUp
	r.EleDown = o.eleDown
	r.EleMax = o.eleMax
	r.EleMin = o.eleMin
	r.EleUpGPX = o.eleUpGPX
	r.EleDownGPX = o.eleDownGPX
	r.EleMissing = o.eleMissing
	r.EleMean = o.EleMean
	r.EleLevelled = o.filter.eleLeveled
	r.Ipolations = o.filter.ipolations
	r.Levelations = o.filter.levelations
	r.FilterRounds = o.filter.ipoRounds
	r.JriderTotal = o.JouleRider
	r.Time = o.TimeRider
	r.TimeTargetSpeeds = o.TimeTarget
	r.JfromTargetPower = o.JriderTarget
	r.LatMean = o.LatMean
	r.Gravity = o.Gravity
	r.Filtered = o.eleUpGPX - o.eleUp
	r.Filterable = max(o.eleUpGPX, o.eleDownGPX) - (o.eleMax - o.eleMin) // max & min should be from GPX
	r.FilteredPros = 100 * r.Filtered / r.Filterable
}

func (r *Results) addRoadSegment(s *segment, p par) {

	if s.distHor <= distTOL {
		return
	}
	if s.jouleRider > 0 {
		r.addRider(s, p)
	}
	r.addJoules(s)
	r.addDists(s)
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
	// r.TimeTargetSpeeds += s.dist / s.vTarget
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

func (r *Results) addDists(s *segment) {

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

	jouleNetSum := s.jouleRider + s.jouleBraking + s.jouleRoll //+ s.jouleSink
	jouleNetSum += jouleDrag + jouleGrav + jouleKinetic
	r.SegEnergyMeanAbs += math.Abs(jouleNetSum)
	r.SegEnergyMean += jouleNetSum

	r.Jroll += s.jouleRoll
	r.Jsink += s.jouleSink
	r.Jbraking += s.jouleBraking

	if jouleDrag > 0 {
		r.JdragPush += jouleDrag
	} else {
		r.JdragRider += s.jouleDragRider
		r.JdragBrake += s.jouleDragBrake
		r.JdragFreewheel += s.jouleDragFreewh
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
	// if s.powerTarget > 0 {
	// 	r.JfromTargetPower += s.dist * s.powerTarget / s.vTarget
	// }
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
	r.JriderTotal *= p.PowerOut      // here, not in unitConversionOut
	r.JfromTargetPower *= p.PowerOut // overestimates r.JriderTotal 2-5%, only
	r.PowerRiderAvg = r.JriderTotal / (r.Time - r.TimeBraking)

	r.BananaRider = r.JriderTotal * j2banana
	r.FatRider = r.JriderTotal * j2lard
	r.FoodRider = r.JriderTotal * j2kcal
	r.JlossDT = r.JriderTotal * p.Bike.DrivetrainLoss / 100

	// Kinetic energy gained by acceleration is not lost. It is later used
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
	j += r.Jsink
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

// unitConversionOut
func (r *Results) unitConversionOut() {
	if r.WindCourse > 0 {
		r.WindCourse *= (180 / π)
	}
	r.DistTotal *= m2km
	r.DistDirect *= m2km
	r.DistLine *= m2km
	r.DistGPX *= m2km
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

	//J* is from now on Wh* *************************
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
