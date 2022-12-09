package route

import "math"

func (o *Route) Filter() {

	prev := 0
	f := &o.filter
	gLim := f.initRelgrade
	gDecrease := 0.5

	if f.rounds < 1 {
		if o.filter.levelFactor > 0 {
			o.filterLevel()
		}
		return
	}
	if f.rounds == 1 {
		gLim = min(0.035, gLim)
	}
	if f.rounds > 2 {
		gDecrease = 0.6
	}
	if f.ipoSumDist < 4*o.distMedian {
		f.ipoSumDist = max(150, 4*o.distMedian)
	}
	if f.ipoDist < 1.1*o.distMedian {
		f.ipoDist = max(25, 1.1*o.distMedian)
	}
	o.prepareIpo()

	for i := 1; i <= f.rounds; i++ {
		
		o.filterBackSteps(gLim)
		// o.filterBackStepDist(gLim)

		if gLim *= gDecrease; gLim < f.minRelGrade {
			gLim = f.minRelGrade
		}
		if prev == f.ipolations {
			f.rounds = i
			break
		}
		prev = f.ipolations
	}
	if o.filter.levelFactor > 0 {
		o.filterLevel()
	}
	if false {
		o.filterBackStepDist(gLim)
	}
	o.recalcDist()
}

func (o *Route) prepareIpo() {

	r := o.route
	f := o.filter
	prevDist := r[1].distHor

	r[1].interpolable = false

	for i := 2; i <= o.segments; i++ {
		s := &r[i]
		dist := s.distHor
		distSum := dist + prevDist

		s.ipoCoef = 1 / distSum
		s.interpolable = prevDist < f.ipoDist || dist < f.ipoDist || distSum < f.ipoSumDist
		prevDist = dist
	}
}

func (o *Route) recalcDist() {
	r := o.route
	for i := 1; i <= o.segments; i++ {
		s := &r[i]
		s.dist = s.distHor * math.Sqrt(1+s.grade*s.grade)
	}
}

func (r route) interpolate(j int) {
	I, J, K := &r[j-1], &r[j], &r[j+1]

	I.grade = (K.ele - I.ele) * J.ipoCoef
	J.grade = I.grade
	J.ele = I.ele + I.grade*I.distHor
}

func (o *Route) filterLevel() {
	f := &o.filter
	if f.levelMax <= 0 || f.levelMax > f.maxFilteredEle {
		f.levelMax = f.maxFilteredEle
	}
	for j := 2; j <= o.segments; j++ {
		o.level(j)
	}
}

func (o *Route) level(j int) {

	I, J := &o.route[j-1], &o.route[j]
	if I.grade*J.grade > -0.005*0.005 {
		return 
	}
	K := &o.route[j+1]
	max, min := I.ele, K.ele
	if max < min {
		max, min = min, max
	}
	
	f := &o.filter
	f.levelations++

	dEle := 0.0
	switch {
	case J.ele > max:
		dEle = (max - J.ele)
		if -dEle < f.levelMin {
			break
		}
		dEle *= f.levelFactor
		if dEle < -f.levelMax {
			dEle = -f.levelMax
		}
	case J.ele < min:
		dEle = (min - J.ele)
		if dEle < f.levelMin {
			break
		}
		dEle *= f.levelFactor
		if dEle > f.levelMax {
			dEle = f.levelMax
		}
	}
	J.ele += dEle
	I.grade = (J.ele - I.ele) / I.distHor
	J.grade = (K.ele - J.ele) / J.distHor
	f.eleLeveled +=  math.Abs(dEle)
}

func (o *Route) filterBackSteps(gradeLim float64) {
	var (
		ipo, backsteps = 0, 0
		enough         = 5 * o.segments
		r              = o.route
		backstepLim    = o.filter.backsteps
		segments       = o.segments
		maxFiltered    = o.filter.maxFilteredEle
	)
	r[1].interpolable = false

	for j := 2; j <= segments; j++ {

		if !r[j].interpolable {
			backsteps = 0
			continue
		}
		if math.Abs(r[j-1].grade-r[j].grade) < gradeLim {
			backsteps = 0
			continue
		}
		if math.Abs(r[j].ele-r[j].eleGPX) > maxFiltered {
			backsteps = 0
			continue
		}
		r.interpolate(j)
		ipo++
		if backsteps < backstepLim {
			j -= 2
			backsteps++
		}
		if ipo > enough {
			break
		}
	}
	o.filter.ipolations += ipo
}

func (o *Route) filterBackStepDist(gradeLim float64) {
	const big = int(1e+9)
	var (
		maxi, ipo, right = 0, 0, 0
		dist             = 0.0
		left             = big
		enough           = 10 * o.segments
		r                = o.route
		backstepDist     = o.filter.backstepDist
		segments         = o.segments
		maxFiltered    = o.filter.maxFilteredEle
	)
	r[1].interpolable = false

	for i := 1; i < segments; i++ {
		j := i + 1
		if maxi < i {
			maxi = i
		}
		if !r[j].interpolable || math.Abs(r[j].ele-r[j].eleGPX) > maxFiltered {
			continue
		}
		if math.Abs(r[i].grade-r[j].grade) < gradeLim {
			if i == maxi {
				//new segment and no interpolation -> reset counters
				left, right, dist = big, 0, 0
			}
			continue
		}
		r.interpolate(j)
		ipo++
		if j < left || j > right {
			if j > right {
				right = j
			} else {
				left = j
			}
			dist += r[j].distHor
			if dist > backstepDist {
				i = maxi
				continue //to next new segment
			}
		}
		i -= 2 //backstep to previous segment

		if ipo > enough {
			//here with long backstepDist and small initial gradeLim
			break
		}
	}
	o.filter.ipolations += ipo
}
