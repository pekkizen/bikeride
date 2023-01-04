package route

import "math"

func (o *Route) Filter() {
	f := &o.filter

	if f.movingAvgWeight > 0 {
		o.movingAverage()
	}
	if f.rounds < 1 {
		if f.levelFactor > 0 {
			o.filterLevel()
		}
		return
	}
	o.setupInterpolation()
	o.filterInterpolate()

	if f.levelFactor > 0 {
		o.filterLevel()
	}
	o.recalcDistances()
}

func (o *Route) filterInterpolate() {

	var (
		f           = &o.filter
		gLim        = f.initRelgrade
		minRelGrade = f.minRelGrade
		prev        = 0
		gDecrease   = 0.5
	)
	if minRelGrade <= 0 {
		minRelGrade = 0.001
	}
	if gLim <= 0 {
		gLim = 0.08
		if f.rounds == 1 {
			gLim = 0.04
		}
	}
	if f.rounds > 2 {
		gDecrease = 2.0 / 3
	}
	for i := 1; i <= f.rounds; i++ {

		o.interpolateWithBackSteps(gLim)

		if gLim *= gDecrease; gLim < minRelGrade {
			gLim = minRelGrade
		}
		if prev == f.ipolations {
			f.rounds = i
			break
		}
		prev = f.ipolations
	}
}

func (o *Route) setupInterpolation() {
	var (
		ipoDist    = o.filter.ipoDist
		ipoSumDist = o.filter.ipoSumDist
		r          = o.route
		prevDist   = r[1].dist
	)
	if ipoDist <= 0 {
		ipoDist = min(40, 1.25*o.distMedian)
	}
	if ipoSumDist <= 0 {
		ipoSumDist = min(175, 5*ipoDist)
	}
	r[1].ipolable = false

	for i := 2; i <= o.segments; i++ {
		J := &r[i]
		dist := J.distHor
		distSum := dist + prevDist
		J.ipoDenominator = 1 / distSum
		J.ipolable = prevDist < ipoDist || dist < ipoDist || distSum < ipoSumDist
		prevDist = dist
	}
}

func (o *Route) movingAverage() {
	var (
		weight = o.filter.movingAvgWeight * o.distMedian
		r      = o.route
		I      = &r[1]
	)
	for i := 2; i <= o.segments; i++ {
		J := &r[i]

		distHor := I.distHor
		w := weight / distHor
		J.ele = (J.ele + w*I.ele) / (w + 1)
		dEle := J.ele - I.ele
		I.grade = dEle / distHor
		I.dist = math.Sqrt(dEle*dEle + distHor*distHor)
		I = J
	}
}

func (o *Route) recalcDistances() {
	r := o.route

	for i := 1; i <= o.segments; i++ {
		s := &r[i]
		s.dist = s.distHor * math.Sqrt(1+s.grade*s.grade)
	}
}

func (o *Route) filterLevel() {

	var (
		f = &o.filter
		I *segment
		J = &o.route[1]
		K = &o.route[2]
	)
	if f.levelMax <= 0 || f.levelMax > f.maxFilteredEle {
		f.levelMax = f.maxFilteredEle
	}
	for k := 3; k <= o.segments; k++ {

		I, J = J, K
		K = &o.route[k]

		if I.grade*J.grade >= 0 || math.Abs(J.ele-J.eleGPX) > f.maxFilteredEle {
			continue
		}
		f.level(I, J, K)
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
		dEle = -f.levelMin + (dEle + f.levelMin) * f.levelFactor
		// dEle *= f.levelFactor
		if dEle < -f.levelMax {
			dEle = -f.levelMax
		}
	case J.ele < min:
		dEle = (min - J.ele)
		if dEle < f.levelMin {
			break
		}
		dEle = f.levelMin + (dEle - f.levelMin) * f.levelFactor
		// dEle *= f.levelFactor
		if dEle > f.levelMax {
			dEle = f.levelMax
		}
	}
	J.ele += dEle
	I.grade = (J.ele - I.ele) / I.distHor
	J.grade = (K.ele - J.ele) / J.distHor
	f.eleLeveled += math.Abs(dEle)
	f.levelations++
	if f.rounds > 0 { //distances recalced later
		return
	}
	I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	J.dist = J.distHor * math.Sqrt(1+J.grade*J.grade)
}

func interpolate(I, J, K *segment) {

	I.grade = (K.ele - I.ele) * J.ipoDenominator
	J.grade = I.grade
	J.ele = I.ele + I.grade*I.distHor
}

func (o *Route) interpolateWithBackSteps(gradeLim float64) {
	var (
		ipo, backsteps = 0, 0
		enough         = 5 * o.segments
		r              = o.route
		stepLim        = o.filter.backsteps
		maxFiltered    = o.filter.maxFilteredEle
		maxIpo         = o.filter.maxIpolations
	)
	r[1].ipolable = false

	for j := 2; j <= o.segments; j++ {

		I, J, K := &r[j-1], &r[j], &r[j+1]

		if math.Abs(I.grade-J.grade) < gradeLim || !J.ipolable {
			backsteps = 0
			continue
		}
		interpolate(I, J, K)

		J.ipolations++
		J.ipolable = math.Abs(J.ele-J.eleGPX) < maxFiltered && J.ipolations < maxIpo
		ipo++
		if backsteps < stepLim {
			j -= 2
			backsteps++
		}
		if ipo > enough {
			break
		}
	}
	o.filter.ipolations += ipo
}
