package route

import (
	// "fmt"
	"math"
)

func (o *Route) Filter() {
	f := &o.filter

	if f.weightedAvgWeight <= 0 && 
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
	if f.weightedAvgWeight > 0 {
		o.filterWeightedAverage()
	}
	if f.levelFactor > 0 {
		o.filterLevel()
	}
	if f.maxAcceptedGrade > 0 {
		o.filterGradientReduce()
	}
	o.recalcDistances()
}

func (o *Route) filterInterpolate() {
	var (
		f           = &o.filter
		gradeLim    = f.initRelgrade
		minRelGrade = f.minRelGrade
		prevIpo     = 0
		decr       = 0.5
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

func (o *Route) filterWeightedAverage() {
	var (
		weight = o.filter.weightedAvgWeight * o.distMedian
		r      = o.route
		I      = &r[o.segments]
	)
	for k := o.segments-1; k > 0; k-- {
		J := &r[k]
		w := weight / I.distHor
		J.ele = (J.ele + w*I.ele) / (w + 1)
		I = J
	}
	I = &r[1]
	for k := 2; k <= o.segments+1; k++ {
		J := &r[k]
		w := weight / I.distHor
		J.ele = (J.ele + w*I.ele) / (w + 1)
		I.grade = (J.ele - I.ele) / I.distHor
		// I.dist = I.distHor*math.Sqrt(1 + I.grade*I.grade)
		I = J
	}
}

func (o *Route) recalcDistances() {
	r := o.route
	for i := 1; i <= o.segments; i++ {
		s := &r[i]
		// s.grade = (r[i+1].ele - s.ele) / s.distHor
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

		I, J = J, K
		K = &r[k]

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
		ipoDist = min(40, 1.5*o.distMedian)
	}
	if ipoSumDist <= 0 {
		ipoSumDist = min(175, 5*ipoDist)
	}
	r[1].ipolable = false

	for i := 2; i <= o.segments; i++ {
		J := &r[i]
		dist := J.distHor
		J.overDistHorSum = 1 / (dist + prevDist)
		J.ipolable = prevDist < ipoDist || dist < ipoDist || (dist+prevDist) < ipoSumDist
		prevDist = dist
	}
}

func interpolate(I, J, K *segment) {

	I.grade = (K.ele - I.ele) * J.overDistHorSum
	J.grade = I.grade
	J.ele = I.ele + I.grade*I.distHor
	J.ipolations++
}

func (o *Route) interpolateWithBackSteps(gradeLim float64) {
	if o.filter.backsteps < 0 {
		return
	}
	const maxIpolations = 15
	var (
		ipo, backsteps int
		enough         = 5 * o.segments
		stepLim        = o.filter.backsteps
		r              = o.route
		gradeLim2      = max(0.015, gradeLim/4)
	)
	r[1].ipolable = false

	for j := 2; j <= o.segments; j++ {

		I, J, K := &r[j-1], &r[j], &r[j+1]

		relGrade := math.Abs(I.grade - J.grade)
		if relGrade < gradeLim || !J.ipolable {

			if relGrade > gradeLim2 {
				interpolate(I, J, K)
			}
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
		if J.ipolations > maxIpolations {
			J.ipolable = false
		}
	}
	o.filter.ipolations += ipo
}

func (o *Route) interpolateDistance(i, j int, dist float64) {
	var (
		r    = o.route
		s    = &r[j+1]
		next = &r[i]
	)
	// if dist == 0 {
	// 	for k := i; k <= j; k++ {
	// 		dist += r[k].distHor
	// 	}
	// }
	grade := (s.ele - next.ele) / dist
	for k := i; k < j; k++ {
		s, next = next, &r[k+1]
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
		if eleUp == 0  {
			left = right - 1
			continue
		}
		elePlain := r[right+1].ele - r[left].ele

		if elePlain > 0 {
			eleUp -= elePlain
		}
		if eleUp > dist*gradeLim {
			o.interpolateDistance(left, right, dist)
			left = right - 1
			continue
		}
		left += 2 * (right - left) / 3
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
				break
			}
			right++
		}
		left = right + 1
	}
	// may leave the last ones unchanged
}
