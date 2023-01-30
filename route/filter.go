package route

import "math"

func (o *Route) Filter() {
	f := &o.filter

	if f.smoothingWeight == 0 &&
		f.ipoBackStepRounds <= 0 &&
		f.levelFactor <= 0 &&
		f.distFilterGrade <= 0 &&
		f.maxAcceptedGrade <= 0 {
		return
	}
	if f.distFilterGrade > 0 {
		o.filterDistanceInterpolation()
	}
	if f.ipoBackStepRounds > 0 {
		o.filterInterpolate()
	}
	if f.smoothingWeight > 0 {
		o.filterWeightedExponential()
	}
	if f.levelFactor > 0 {
		o.filterLevel()
	}
	if f.maxAcceptedGrade > 0 {
		o.filterGradientReduce()
	}
	o.recalcDistances(1, o.segments)
}

func (o *Route) filterInterpolate() {
	var (
		f           = &o.filter
		gradeLim    = f.initRelgrade
		minRelGrade = f.minRelGrade
		prevIpo     = 0
		decr        = 0.5
	)
	if minRelGrade <= 0 {
		minRelGrade = 0.001
	}
	if gradeLim <= 0 {
		gradeLim = 0.08
		if f.ipoBackStepRounds == 1 {
			gradeLim = 0.05
		}
	}
	if f.ipoBackStepRounds > 2 {
		decr = 2.0 / 3
	}
	o.setupInterpolation()

	for i := 1; i <= f.ipoBackStepRounds; i++ {

		o.interpolateWithBackSteps(gradeLim)

		if gradeLim *= decr; gradeLim < minRelGrade {
			gradeLim = minRelGrade
		}
		if prevIpo == f.ipolations {
			f.ipoBackStepRounds = i
			break
		}
		prevIpo = f.ipolations
	}
}

// https://en.wikipedia.org/wiki/Exponential_smoothing#Basic_(simple)_exponential_smoothing
// filterWeightedExponential implements simple exponential smoothing by using
// inverse road segment length as a factor of the smoothing weight.

func (o *Route) filterWeightedExponential() {
	var (
		weight = o.filter.smoothingWeight * o.distMedian
		r      = o.route
		J      = &r[o.segments]
		I      *segment
	)
	for k := o.segments - 1; k > 0; k-- {
		I, J = J, &r[k]
		w := weight / I.distHor
		J.ele = (J.ele + w*I.ele) / (w + 1)
	}
	J = &r[1]
	for k := 2; k <= o.segments+1; k++ {
		I, J = J, &r[k]
		w := weight / I.distHor
		J.ele = (J.ele + w*I.ele) / (w + 1)

		I.grade = (J.ele - I.ele) / I.distHor
		// I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	}
}

func (o *Route) recalcDistances(left, right int) {
	r := o.route
	for ;left <= right; left++ {
		s := &r[left]
		s.dist = s.distHor * math.Sqrt(1+s.grade*s.grade)
		
	}
}

func (o *Route) filterLevel() {
	var (
		r    = o.route
		I    *segment
		J, K = &r[1], &r[2]
	)
	for k := 3; k <= o.segments; k++ {

		I, J, K = J, K, &r[k]

		if I.grade*J.grade >= 0 {
			continue
		}
		o.filter.level(I, J, K)
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
	// I.dist = I.distHor * math.Sqrt(1+I.grade*I.grade)
	// J.dist = J.distHor * math.Sqrt(1+J.grade*J.grade)
	f.eleLeveled += math.Abs(dEle)
	f.levelations++
}

func (o *Route) setupInterpolation() {
	var (
		ipoDist    = o.filter.ipoDist
		ipoSumDist = o.filter.ipoSumDist
		r          = o.route
		prevDist   = r[1].dist
	)
	if ipoDist <= 0 {
		ipoDist = min(40, 2*o.distMedian)
	}
	if ipoSumDist <= 0 {
		ipoSumDist = min(200, 5*ipoDist)
	}
	r[1].ipolable = false

	for i := 2; i <= o.segments; i++ {
		s := &r[i]
		dist := s.distHor
		s.ipolable = prevDist < ipoDist || dist < ipoDist || dist+prevDist < ipoSumDist
		prevDist = dist
	}
}

func interpolate(I, J, K *segment) {

	grade := (K.ele - I.ele) / (I.distHor + J.distHor)
	J.grade = grade
	I.grade = grade
	J.ele = I.ele + grade*I.distHor
	J.ipolations++
}

func (o *Route) interpolateWithBackSteps(gradeLim float64) {
	const maxIpolations = 15
	var (
		ipo, backsteps int
		enough         = 5 * o.segments
		stepLim        = o.filter.backsteps
		r              = o.route
	)
	for j := 2; j <= o.segments; j++ {

		I, J, K := &r[j-1], &r[j], &r[j+1]

		if math.Abs(I.grade-J.grade) < gradeLim || !J.ipolable {
			backsteps = 0
			continue
		}
		interpolate(I, J, K)

		if backsteps < stepLim {
			j -= 2
			backsteps++
		}
		if ipo++; ipo > enough {
			break
		}
		if J.ipolations >= maxIpolations {
			J.ipolable = false
		}
	}
	o.filter.ipolations += ipo
}

func (o *Route) interpolateDistance(i, j int, dist float64) {
	var (
		r     = o.route
		next  = &r[i]
		grade = (r[j+1].ele - next.ele) / dist
	)
	for k := i; k < j; k++ {
		s := next
		next = &r[k+1]
		s.grade = grade
		next.ele = s.ele + s.distHor*grade
		s.ipolations++
	}
	next.grade = grade

	next.ipolations++
	o.filter.ipolations += j - i + 1
}

func (o *Route) filterDistanceInterpolation() {
	var (
		ipoDist       = o.filter.ipoSumDist
		gradeLim      = o.filter.distFilterGrade
		last          = o.segments
		r             = o.route
		left, right   = 1, 1
		dEle, distHor float64
	)
	if ipoDist <= 0 {
		ipoDist = 200
	}
	for left < last-2 {
		dist, eleUp := 0.0, 0.0

		for right = left; dist < ipoDist && right <= last; right++ {
			s := &r[right]
			distHor = s.distHor
			dist += distHor
			dEle = s.grade * s.distHor
			if dEle > 0 {
				eleUp += dEle
			}
		}
		right--

		if dist > 1.5*ipoDist {
			dist -= distHor
			if dEle > 0 {
				eleUp -= dEle
			}
			right--
		}
		if right-left < 2 {
			left++
			continue
		}
		elePlain := r[right+1].ele - r[left].ele

		if elePlain > 0 {
			eleUp -= elePlain
		}
		if eleUp < dist*gradeLim {
			left += 2 * (right - left) / 3
			continue
		}
		o.interpolateDistance(left, right, dist)

		if right -= 2; left < right {
			left = right
			continue
		}
		left++
	}
}

func (o *Route) filterGradientReduce() {
	var (
		last     = o.segments
		r        = o.route
		limGrade = o.filter.maxAcceptedGrade
		left     = 1
	)
	for left < last {

		for math.Abs(r[left].grade) < limGrade && left < last {
			left++
		}
		var (
			eleStart = r[left].ele
			dist     = r[left].dist
			right    = left + 1
		)
		for right <= last {

			dist += r[right].dist
			longGrade := (r[right+1].ele - eleStart) / dist

			if math.Abs(longGrade) < limGrade {
				o.interpolateDistance(left, right, dist)
				// o.recalcDistances(left, right)
				break
			}
			right++
		}
		left = right + 1
	}
	// may leave the last ones unchanged
}
