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
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// func minMax(x, minv, maxv float64) (mi, ma float64) {
// 	return min(x, minv), max(x, maxv)
// }

func (o *Route) Results(c *motion.BikeCalc, p par, l *logerr.Logerr) *Results {

	r := &Results{}
	rou := o.route[1 : len(o.route)-1]

	r.calcRouteStats(o)
	o.calcRouteCourse()
	r.addRouteStats(o, p)
	r.addCalculatorStats(c)

	for i := range rou {
		r.addRoadSegment(&rou[i], p)
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
		eleSum         [2]float64
		distsum        float64
		sumGradeChange float64
		prevDist       = o.distMedian
		prevGrade      float64
		maxGrade       = -9999.0
		minGrade       = 9999.0
		minDist        = 9999.0
		maxDist        = 0.0
		minVel         = 9999.0
		maxVel         = 0.0
		rou            = o.route[1 : len(o.route)-1]
		maxEle         = max(rou[0].ele, rou[len(rou)/2].ele, rou[len(rou)-1].ele)
		minEle         = min(rou[0].ele, rou[len(rou)/2].ele, rou[len(rou)-1].ele)
	)
	for i := range rou {
		s := &rou[i]
		grade, dist, vExit, ele := s.grade, s.dist, s.vExit, s.ele
		dEle := s.grade * s.distHor
		// grade change / 1 meter / two half road segments
		sumGradeChange += 2 / (dist + prevDist) * math.Abs(grade-prevGrade)
		prevGrade = grade
		prevDist = dist
		distsum += dist
		eleSum[signbit(dEle)] += dEle

		// minGrade = min(grade, minGrade)
		// maxGrade = max(grade, maxGrade)
		// minDist = min(dist, minDist)
		// maxDist = max(dist, maxDist)
		// minVel = min(vExit, minVel)
		// maxVel = max(vExit, maxVel)
		// minEle = min(ele, minEle)
		// maxEle = max(ele, maxEle)

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
		if vExit < minVel {
			minVel = vExit
		}
		if vExit > maxVel {
			maxVel = vExit
		}
		if ele < minEle {
			minEle = ele
		}
		if ele > maxEle {
			maxEle = ele
		}

	}
	o.distance = distsum
	o.eleUp = eleSum[0]
	o.eleDown = -eleSum[1]
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
	r.VelMax = maxVel
	r.VelMin = minVel
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

	r.JgravDown = r.jouleG[0]
	r.JgravUp = r.jouleG[1]
	r.JkineticDece = r.jouleK[0]
	r.JkineticAcce = r.jouleK[1]

	if r.CalcSegs > r.Segments/10 {
		r.CalcStepsAvg = float64(r.CalcSteps) / float64(r.CalcSegs)
		r.SingleStepPros /= float64(r.CalcSegs)
		r.SingleStepPros *= 100
	}
	r.VelAvg = r.DistTotal / r.Time
	if r.TimeDownhill > 0 {
		r.VelDownVert = r.VerticalDownEle * 3600 / r.TimeDownhill
		r.VelDownhill = r.DistDownhill / r.TimeDownhill
	}
	c.SetWind(0)
	t := &p.Ride
	q := &p.Powermodel

	if t.BrakingDist <= 0 {
		r.DownhillMaxSpeed = (t.VerticalDownSpeed / -dhSpeedGRADE) * mh2ms * ms2kmh
	} else {
		c.SetGradeExact(dhSpeedGRADE)
		r.DownhillMaxSpeed = c.MaxBrakeStopVelNoWind(t.BrakingDist) * ms2kmh
	}
	if t.MinSpeed > 0 {
		r.MaxGradeUp = c.GradeFromVelAndPower(t.MinSpeed, q.UphillPower) * 100
	}
	c.SetGradeExact(q.DownhillPowerGrade)
	r.DownhillPowerSpeed, _ = c.NewtonRaphson(q.DownhillPower*0.01*q.FlatPower, minTolNR, 8)
	r.DownhillPowerSpeed *= ms2kmh
	r.RhoBase, _ = c.RhoFromEle(p.Environment.BaseElevation)
}

func (r *Results) addCalculatorStats(c *motion.BikeCalc) {
	r.VelErrorMean = c.VelErrorMean()
	r.VelErrorAbsMean = c.VelErrorAbsMean()
	r.VelErrorPos = c.VelErrorPos()
	r.VelErrorMax = c.VelErrorMax()
	r.PowerErrors = c.PowerErrors()
	r.VelTol = c.VelTol()
	r.SolverRoundsAvg = float64(c.SolverRounds()) / max(1, float64(c.SolverCalls()))
	r.SolverCalls = c.SolverCalls()
	r.MaxIter = c.MaxIter()
	r.Counter = c.Counter()
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
	r.Time = o.Time
	r.TimeTargetSpeeds = o.TimeTarget
	r.JfromTargetPower = o.JriderTarget
	r.LatMean = o.LatMean
	r.Gravity = o.Gravity
	r.Filtered = o.eleUpGPX - o.eleUp
	r.Filterable = max(o.eleUpGPX, o.eleDownGPX) - (o.eleMax - o.eleMin) // max & min should be from GPX
	r.FilteredPros = 100 * r.Filtered / r.Filterable
}

func (r *Results) addRoadSegment(s *segment, p par) {
	if s.distHor <= distTol {
		return
	}
	if s.jouleRider > 0 {
		r.addRider(s, p)
	}
	r.addJoules(s)
	r.addDists(s)
	r.addEleUpByMomentum(s)
	r.TimeUHBreaks += s.timeBreak
	r.TimeBraking += s.timeBrake
	r.DistFreewheel += s.distFreewheel
	r.TimeFreewheel += s.timeFreewheel
	r.CalcSteps += s.calcSteps
	r.CalcSegs++
	if s.calcSteps == 1 {
		r.SingleStepPros++
	}
}

func (r *Results) addEleUpByMomentum(s *segment) {
	if s.grade <= 0 || s.jouleKinetic <= 0 {
		return
	}
	JwindPush := max(0, s.jouleDrag)
	JkineticShare := s.jouleKinetic / (s.jouleKinetic + s.jouleDeceRider + JwindPush)
	r.EleUpKinetic += JkineticShare * (s.grade * s.distKinetic) // shoud be sin for grade=tan
}

func (r *Results) addDists(s *segment) {
	r.DistBrake += s.distBrake
	r.DistRider += s.distRider

	grade, dist := s.grade, s.dist
	switch {
	case math.Abs(grade) < flatGRADE:
		r.DistFlat += dist

	case grade >= uphillGRADE:
		r.DistUphill += dist

	case grade <= downhillGRADE:
		r.TimeDownhill += s.time
		r.DistDownhill += dist
		r.VerticalDownEle -= s.distHor * grade
	}
}

func (r *Results) addJoules(s *segment) {

	jouleDrag, jouleGrav, jouleKinetic := s.jouleDrag, s.jouleGrav, s.jouleKinetic

	jouleNetSum := s.jouleRider + s.jouleBrake + s.jouleRoll //+ s.jouleSink
	jouleNetSum += jouleDrag + jouleGrav + jouleKinetic
	r.SegEnergyMeanAbs += math.Abs(jouleNetSum)
	r.SegEnergyMean += jouleNetSum
	s.jouleNetSum = jouleNetSum // seg s updated here

	r.Jroll += s.jouleRoll
	r.Jsink += s.jouleSink
	r.Jbraking += s.jouleBrake

	r.jouleG[signbit(jouleGrav)] += jouleGrav
	r.jouleK[signbit(jouleKinetic)] += jouleKinetic

	if jouleDrag > 0 {
		r.JdragPush += jouleDrag
		return
	}
	r.JdragRider += s.jouleDragRider
	r.JdragBrake += s.jouleDragBrake
	r.JdragFreewheel += s.jouleDragFreewh
	r.JdragResist += jouleDrag

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
		r.JriderDrag += JriderShare * jouleDrag // There is also r.JdragRider
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
	r.JfromTargetPower *= p.PowerOut // overestimates r.JriderTotal ~5%, only
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
	j += r.Jsink // if not zero, something is wrong
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
	r.DistRider *= m2km
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
	r.JdragResist *= j2Wh
	r.Jroll *= j2Wh
	r.JlossDT *= j2Wh
	r.JgravUp *= j2Wh
	r.JgravDown *= j2Wh
	r.Jbraking *= j2Wh
	r.EnergySumRider *= j2Wh
	r.EnergySumTotal *= j2Wh
}
