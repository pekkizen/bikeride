package route

import (
	"math"
)

func (o *Route) Filter() {
	f := &o.filter

	if f.smoothingWeight <= 0 &&
		f.ipoRounds <= 0 &&
		f.levelFactor <= 0 &&
		f.distFilterTol < 0 &&
		f.maxAcceptedGrade <= 0 {
		return
	}
	if f.distFilterTol >= 0 {
		o.filterDistanceShortenInterpolation()
	}
	if f.ipoRounds > 0 {
		o.filterInterpolateWithBackSteps()
		o.recalcRoadDistances(1, o.segments)
	}
	if f.levelFactor > 0 {
		o.filterLevel()
	}
	if f.smoothingWeight > 0 {
		if f.smoothingWeightDist > o.distMedian || f.smoothingWeightDist == -1 {
			f.smoothingWeightDist = o.distMedian
		}
		o.filterWeightedExponential()
	}
	if f.maxAcceptedGrade > 0 {
		o.filterGradientReduce()
	}
	if false && test {
		o.checkDistGradeErrors()
	}
}

// recalcRoadDistances recalculates road distances of segments between left and right.
func (o *Route) recalcRoadDistances(left, right int) {

	r := o.route[left : right+1]

	for k := range r {
		s := &r[k]
		s.dist = s.distHor * math.Sqrt(1+s.grade*s.grade)
	}
}

func (o *Route) checkDistGradeErrors() {
	var (
		s    *segment
		next = &o.route[1]
	)
	for i := 2; i <= len(o.route)-1; i++ {
		s, next = next, &o.route[i]
		grade := (next.ele - s.ele) / s.distHor
		if e := math.Abs(grade - s.grade); e > 2.5e-14 {
			println("Filter grade err:", s.segnum, e, grade)
		}
		dist := s.distHor * math.Sqrt(1+s.grade*s.grade)
		if e := math.Abs(dist-s.dist) / dist; e > 3e-16 {
			println("Filter dist err", s.segnum, e, s.dist)
		}
	}
}

func (o *Route) interpolateWithBackSteps(gradeLim float64) {
	var (
		ipo, backsteps int
		enough         = 5 * o.segments
		stepLim        = o.filter.backsteps
		r              = o.route
		ipoDist        = o.filter.ipoDist
		ipoSumDist     = o.filter.ipoSumDist
	)
	for k := 2; k <= len(r)-2; k++ {

		I, J, K := &r[k-1], &r[k], &r[k+1]

		if I.distHor > ipoDist &&
			J.distHor > ipoDist &&
			I.distHor+J.distHor > ipoSumDist {
			backsteps = 0
			continue
		}
		if math.Abs(I.grade-J.grade) < gradeLim {
			backsteps = 0
			continue
		}
		J.interpolate(I, K)

		if backsteps < stepLim && k > 2 {
			k -= 2
			backsteps++
		}
		if ipo++; ipo > enough {
			break
		}
	}
	o.filter.ipolations += ipo
}

// interpolate interpolates elevation of segment J from
// elevations of segments I and K.
func (J *segment) interpolate(I, K *segment) {

	tan := (K.ele - I.ele) / (I.distHor + J.distHor)
	J.grade = tan
	I.grade = tan
	J.ele = I.ele + tan*I.distHor
}

func (o *Route) filterInterpolateWithBackSteps() {
	var (
		f           = &o.filter
		gradeLim    = f.initRelgrade
		minRelGrade = f.minRelGrade
		prevIpo     = 0
		decrefactor = 0.5
	)
	if minRelGrade <= 0 {
		minRelGrade = 0.001
	}
	if gradeLim <= 0 {
		gradeLim = 0.07
		if f.ipoRounds == 1 {
			gradeLim = 0.04
		}
	}
	if f.ipoRounds > 3 {
		decrefactor = 2.0 / 3
	}
	for i := 1; i <= f.ipoRounds; i++ {

		o.interpolateWithBackSteps(gradeLim)

		if gradeLim *= decrefactor; gradeLim < minRelGrade {
			gradeLim = minRelGrade
		}
		if prevIpo == f.ipolations && i > 10 {
			f.ipoRounds = i
			break
		}
		prevIpo = f.ipolations
	}
}

// https://en.wikipedia.org/wiki/Exponential_smoothing#Basic_(simple)_exponential_smoothing

// filterWeightedExponential implements simple exponential smoothing by using
// inverse distance between elevation points as factor of the smoothing weight.
// The smoothing is done backwards and forwards, so that the smoothed elevation
// profile keeps on the "top" of the original profile. Linear sections of the
// original elevation profile are only very lightly changed. Second round returns the
// elevations back to near their original values.
func (o *Route) filterWeightedExponential() {
	var (
		weight = o.filter.smoothingWeight * o.filter.smoothingWeightDist
		r      = o.route
		I, J   *segment
	)
	I = &r[len(r)-2]
	for k := len(r) - 3; k > 1; k-- { //backwards
		I, J = &r[k], I

		w := weight / (I.distHor + weight)
		I.ele = I.ele + w*(J.ele-I.ele)
	}
	J = &r[1]
	for k := 2; k < len(r); k++ { //forwards
		I, J = J, &r[k]

		w := weight / (I.distHor + weight)
		J.ele = J.ele + w*(I.ele-J.ele)

		I.grade = (J.ele - I.ele) / I.distHor
		I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	}
}

// filterLevel lowers down single high points and lifts up single low points.
// It uses Route.filter parameters levelFactor, levelMin and levelMax to
// calculate the amount of leveling (elevation change) made.
func (o *Route) filterLevel() {
	var (
		r    = o.route
		I    *segment
		J, K = &r[1], &r[2]
	)
	for k := 3; k < len(r)-1; k++ {
		I, J, K = J, K, &r[k]

		if I.grade*J.grade < 0 && math.Abs(I.grade-J.grade) < 0.01 {
			o.filter.level(I, J, K)
		}
	}
}

func (f *filter) level(I, J, K *segment) {

	hi := max(I.ele, K.ele)
	lo := min(I.ele, K.ele)
	dEle := 0.0
	switch {
	case J.ele > hi:
		dEle = (hi - J.ele)
		if -dEle < f.levelMin {
			break
		}
		dEle = -f.levelMin + (dEle+f.levelMin)*f.levelFactor
		dEle = max(dEle, f.levelMin)
	case J.ele < lo:
		dEle = (lo - J.ele)
		if dEle < f.levelMin {
			break
		}
		dEle = f.levelMin + (dEle-f.levelMin)*f.levelFactor
		dEle = min(dEle, f.levelMax)
	default:
		return
	}
	J.ele += dEle
	I.grade = (J.ele - I.ele) / I.distHor
	J.grade = (K.ele - J.ele) / J.distHor
	I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	J.dist = J.distHor * math.Sqrt(1+J.grade*J.grade)

	f.eleLeveled += math.Abs(dEle)
	f.levelations++
}

func (o *Route) interpolateDistance(left, right int, tan float64) {
	var (
		r    = o.route[left : right+2]
		next = &r[0]
		s    *segment
	)
	for k := 1; k < len(r)-1; k++ {
		s, next = next, &r[k]
		s.grade = tan
		s.dist = s.distHor * math.Sqrt(1+tan*tan)
		next.ele = s.ele + tan*s.distHor
	}
	tan = (r[len(r)-1].ele - next.ele) / next.distHor
	next.grade = tan
	next.dist = next.distHor * math.Sqrt(1+tan*tan)
	o.filter.ipolations += right - left
}

func (o *Route) filterDistanceShortenInterpolation() {
	var (
		left, right = 1, 1
		r           = o.route
		s           *segment
		distHor     float64
		distIpo     = o.filter.distFilterDist
		tan         = o.filter.distFilterTol
	)
	// Relative excess distance with gradient tan grade/100
	relDistLim := math.Sqrt(1 + tan*tan)
	for left < len(r)-4 {
		distHorSum, distRoad := 0.0, 0.0

		for right = left; distHorSum < distIpo && right < len(r)-1; right++ {
			s = &r[right]
			distHor = s.distHor
			distHorSum += s.distHor
			distRoad += s.dist
		}
		if right--; right-left < 2 {
			left++
			continue
		}
		if right == o.segments && distHorSum < distIpo {
			distIpo = distHorSum
		}
		var (
			eleIpo   = s.ele + s.grade*(distHor-(distHorSum-distIpo))
			dEle     = eleIpo - r[left].ele
			distLine = math.Sqrt(distIpo*distIpo + dEle*dEle)
			nextleft = (left + 2*right) / 3 // backspace 1/3
		)
		if distHor > distIpo || right == o.segments {
			nextleft = right + 1
		}
		distRoad *= distIpo / distHorSum

		if distRoad > distLine*relDistLim {
			o.interpolateDistance(left, right, dEle/distIpo)
		}
		left = nextleft
	}
}

func (o *Route) filterGradientReduce() {
	var (
		r        = o.route
		last     = len(r) - 2
		gradeLim = o.filter.maxAcceptedGrade
		left     = 1
	)
	for left < last {
		for ; left < len(r)-2; left++ {
			if math.Abs(r[left].grade) > gradeLim {
				break
			}
		}
		var (
			eleStart = r[left].ele
			distHor  = r[left].distHor
			right    = left + 1
		)
		for ; right <= last; right++ {

			distHor += r[right].distHor
			tan := (r[right+1].ele - eleStart) / distHor

			if math.Abs(tan) < gradeLim || right == last {
				o.interpolateDistance(left, right, tan)
				break
			}
		}
		left = right
	}
	if math.Abs(r[last].grade) > gradeLim {
		r[last].grade = r[last-1].grade // better solution?
	}
}
