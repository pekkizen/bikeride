package route

import "math"

func (o *Route) Filter() {
	f := &o.filter

	if f.smoothingWeight <= 0 &&
		f.ipoRounds <= 0 &&
		f.levelFactor <= 0 &&
		f.distFilterTol < 0 &&
		f.maxAcceptedGrade <= 0 {
		return
	}
	recalcDists := false
	if f.distFilterTol >= 0 {
		o.filterDistanceShortenInterpolation()
		recalcDists = true
	}
	if f.ipoRounds > 0 {
		o.filterInterpolateWithBackSteps()
		recalcDists = true
	}
	if recalcDists {
		o.recalcRoadDistances(1, o.segments)
	}
	// The following three recalc road distances individually.
	if f.smoothingWeight > 0 {
		if f.smoothingDist < 0 {
			f.smoothingDist = o.distMedian
		}
		o.filterWeightedExponential()
	}
	if f.levelFactor > 0 {
		o.filterLevel()
	}
	if f.maxAcceptedGrade > 0 {
		o.filterGradientReduce()
	}
}

// recalcRoadDistances recalculates road distances of segments between left and right.
func (o *Route) recalcRoadDistances(left, right int) {
	r := o.route
	for ; left <= right; left++ {
		s := &r[left]
		s.dist = s.distHor * math.Sqrt(1+s.grade*s.grade)
		// check errors in grade calculation
		// grade := (r[left+1].ele - s.ele) / s.distHor
		// if math.Abs(grade-s.grade) > 5e-14 {
		// 	println(s.segnum)
		// }
	}
}

// interpolate interpolates elevation of segment J from
// elevations of segments I and K.
func interpolate(I, J, K *segment) {

	tan := (K.ele - I.ele) / (I.distHor + J.distHor)
	J.grade = tan
	I.grade = tan
	J.ele = I.ele + tan*I.distHor
}

func (o *Route) interpolateWithBackSteps(gradeLim float64) {
	var (
		ipo, backsteps int
		enough         = 5 * o.segments
		last           = o.segments
		stepLim        = o.filter.backsteps
		r              = o.route
		ipoDist        = o.filter.ipoDist
		ipoSumDist     = o.filter.ipoSumDist
	)
	for k := 2; k <= last; k++ {

		I, J, K := &r[k-1], &r[k], &r[k+1]

		if math.Abs(I.grade-J.grade) < gradeLim {
			backsteps = 0
			continue
		}
		if I.distHor > ipoDist &&
			J.distHor > ipoDist &&
			I.distHor+J.distHor > ipoSumDist {
			backsteps = 0
			continue
		}
		interpolate(I, J, K)

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
// inverse distance between elevation points as a factor of the smoothing weight.
// The smoothing is done backwards and forwards, so that the smoothed elevation
// profile keeps on the "top" of the original profile. Linear sections of the
// original elevation profile are very lightly changed. Second round returns the
// elevations back to near their original values.
func (o *Route) filterWeightedExponential() {
	var (
		weight = o.filter.smoothingWeight * o.filter.smoothingDist
		r      = o.route
		last   = o.segments
		I, J   *segment
	)
	I = &r[last]
	for k := last - 1; k > 0; k-- { // backwards

		I, J = &r[k], I

		w := weight / I.distHor
		I.ele = (I.ele + w*J.ele) / (w + 1)
	}
	J = &r[1]
	for k := 2; k <= last+1; k++ { // forwards

		I, J = J, &r[k]

		w := weight / I.distHor
		J.ele = (w*I.ele + J.ele) / (w + 1)

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
		last = o.segments
	)
	for k := 3; k <= last; k++ {

		I, J, K = J, K, &r[k]
		if I.grade*J.grade < 0 {
			o.filter.level(I, J, K)
		}
	}
}

func (f *filter) level(I, J, K *segment) {

	max, min := I.ele, K.ele
	if max < min {
		max, min = min, max
	}
	dEle := 0.0
	switch {
	case J.ele > max:
		dEle = (max - J.ele)
		if -dEle < f.levelMin {
			break
		}
		dEle = -f.levelMin + (dEle+f.levelMin)*f.levelFactor
		if dEle < -f.levelMax {
			dEle = -f.levelMax
		}
	case J.ele < min:
		dEle = (min - J.ele)
		if dEle < f.levelMin {
			break
		}
		dEle = f.levelMin + (dEle-f.levelMin)*f.levelFactor
		if dEle > f.levelMax {
			dEle = f.levelMax
		}
	}
	J.ele += dEle
	I.grade = (J.ele - I.ele) / I.distHor
	J.grade = (K.ele - J.ele) / J.distHor
	I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	J.dist = J.distHor * math.Sqrt(1+J.grade*J.grade)

	f.eleLeveled += math.Abs(dEle)
	f.levelations++
}

func (o *Route) interpolateDistance(left, right int, eleIpo, distIpo float64) {
	var (
		r    = o.route
		next = &r[left]
		s    *segment
	)
	tan := (eleIpo - next.ele) / distIpo

	for k := left + 1; k <= right; k++ {

		s, next = next, &r[k]
		s.grade = tan
		next.ele = s.ele + tan*s.distHor
	}
	next.grade = (r[right+1].ele - next.ele) / next.distHor

	o.filter.ipolations += right - left
}

func (o *Route) filterDistanceShortenInterpolation() {
	var (
		distIpo     = o.filter.distFilterDist
		relDistLim  = o.filter.distFilterTol + 1
		left, right = 1, 1
		last        = o.segments
		r           = o.route
		s           *segment
		distHor     float64
	)
	for left < last-2 {
		distHorSum, distRoad := 0.0, 0.0

		for right = left; distHorSum < distIpo && right <= last; right++ {
			s = &r[right]
			distHor = s.distHor
			distHorSum += distHor
			distRoad += s.dist
		}
		if right--; right-left < 2 {
			left++
			continue
		}
		if right == last && distHorSum < distIpo {
			distIpo = distHorSum
		}
		var (
			eleIpo   = s.ele + s.grade*(distHor-(distHorSum-distIpo))
			dEleIpo  = eleIpo - r[left].ele
			distLine = math.Sqrt(distIpo*distIpo + dEleIpo*dEleIpo)
			nextleft = (left + 2*right) / 3
		)
		if distHor > distIpo || right == last {
			nextleft = right + 1
		}
		distRoad *= distIpo / distHorSum

		if distRoad > distLine*relDistLim {
			// o.interpolateDistance(left, (nextleft+right)/2, eleIpo, distIpo)
			o.interpolateDistance(left, right, eleIpo, distIpo)
			// o.recalcRoadDistances(nextleft, right)
		}
		left = nextleft
	}
}

// filterGradientReduce
func (o *Route) filterGradientReduce() {
	var (
		last     = o.segments
		r        = o.route
		gradeLim = o.filter.maxAcceptedGrade
		left     = 1
	)
	for left < last {

		for math.Abs(r[left].grade) < gradeLim && left < last {
			left++
		}
		var (
			eleStart = r[left].ele
			distHor  = r[left].distHor
			right    = left + 1
		)
		for ; right <= last; right++ {

			distHor += r[right].distHor
			eleEnd := r[right+1].ele
			longGrade := (eleEnd - eleStart) / distHor

			if math.Abs(longGrade) < gradeLim || right == last {
				o.interpolateDistance(left, right, eleEnd, distHor)
				o.recalcRoadDistances(left, right)
				break
			}
		}
		left = right
	}
}
