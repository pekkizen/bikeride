package power

import "math"

const (
	simpleExponentialModel = 1
	simpleLinearModel      = 2
	fullExponentialModel   = 3
	fullLinearModel        = 4
	useMathExp             = false
	useFMA                 = true
)

var ln = math.Log

var ratioModel func(*Generator, float64, float64) float64

type Generator struct {
	βU  float64
	βD  float64
	βT  float64
	βH  float64
	βUT float64
	βUH float64
	βDT float64
	βDH float64
	cDT float64
	cUT float64
	cDH float64
	cUH float64

	sysGradeUp   float64
	sysGradeDown float64
	sysTailwind  float64
	sysHeadwind  float64

	expDownhill float64
	expHeadwind float64
	expTailwind float64
	expUphill   float64

	powerDownRatio     float64
	powerTailRatio     float64
	powerHeadRatio     float64
	powerUpRatio       float64
	powerDownHeadRatio float64
	powerDownTailRatio float64
	powerMinRatio      float64

	powerModelType   int
	logPowerUpRatio  float64
	logPowerMinRatio float64
}

func RatioGenerator() *Generator {
	return &Generator{
		cDT: 1,
		cUT: 1,
		cDH: 1,
		cUH: 1,

		sysGradeUp:   0.08,
		sysGradeDown: -0.02,
		sysTailwind:  -5,
		sysHeadwind:  5,

		expDownhill: 2,
		expUphill:   0.75,
		expHeadwind: 1,
		expTailwind: 1,

		powerDownRatio:     0.2,
		powerTailRatio:     0.85,
		powerHeadRatio:     1.15,
		powerUpRatio:       1.70,
		powerDownHeadRatio: 1.0,
		powerDownTailRatio: 0.05,
		powerModelType:     1,
		powerMinRatio:      0.001,
	}
}

func (m *Generator) Setup() {

	if m.powerUpRatio < 1 {
		m.powerUpRatio = 1
	}
	if m.powerDownRatio > 1 {
		m.powerDownRatio = 1
	}
	m.logPowerUpRatio = ln(m.powerUpRatio)
	m.logPowerMinRatio = ln(m.powerMinRatio)

	switch m.powerModelType {
	case simpleExponentialModel:
		m.initSimpleExponential()
		ratioModel = (*Generator).simpleExponential

	case simpleLinearModel:
		m.initSimpleLinear()
		ratioModel = (*Generator).simpleLinear

	case fullExponentialModel:
		m.initFullExponential()
		ratioModel = (*Generator).fullExponential

	case fullLinearModel:
		m.initFullLinear()
		ratioModel = (*Generator).fullLinear
	}
}

func (m *Generator) Ratio(grade, wind float64) (power float64) {

	if grade >= m.sysGradeUp {
		return m.powerUpRatio
	}
	if wind != 0 {
		wind = m.cutWind(grade, wind)
	}
	return ratioModel(m, grade, wind)
}

func (m *Generator) cutWind(grade, wind float64) float64 {
	if wind < m.sysTailwind {
		return m.sysTailwind
	}
	if wind < m.sysHeadwind {
		return wind
	}
	if grade >= m.sysGradeDown {
		return m.sysHeadwind
	}
	//this gives more power to strong head winds and steep downhills.
	return m.sysHeadwind + (wind-m.sysHeadwind)*(grade-m.sysGradeDown)/grade*0.5
}

// Uphill grade function fU = grade^0.75 gives quite linear power increase.
func (m *Generator) initSimpleExponential() {

	// fU := math.Sqrt(m.sysGradeUp * math.Sqrt(m.sysGradeUp)) // grade^0.75
	fU := fU34(m.sysGradeUp) //up ~grade^0.75
	m.βU = ln(m.powerUpRatio) / fU

	fD := -(m.sysGradeDown * m.sysGradeDown)
	m.βD = ln(m.powerDownRatio) / fD

	fT := m.sysTailwind
	m.βT = ln(m.powerTailRatio) / fT

	fH := m.sysHeadwind
	m.βH = ln(m.powerHeadRatio) / fH

	if m.powerDownHeadRatio > 0 {
		m.cDH = m.powerDownHeadRatio / m.powerDownRatio
	}
	if m.powerDownTailRatio > 0 {
		m.cDT = m.powerDownTailRatio / m.powerDownRatio
	}
	m.βUT = (ln(m.cUT) - ln(m.powerTailRatio)) / (fU * fT)
	m.βUH = (ln(m.cUH) - ln(m.powerHeadRatio)) / (fU * fH)
	m.βDT = (ln(m.cDT) - ln(m.powerTailRatio)) / (fD * fT)
	m.βDH = (ln(m.cDH) - ln(m.powerHeadRatio)) / (fD * fH)
}

func (m *Generator) simpleExponential(grade float64, wind float64) float64 {

	var gU, gD, βX float64
	switch {
	case grade > 0:
		// gU = math.Sqrt(grade * math.Sqrt(grade)) // up grade^0.75
		gU = fU34(grade) // approx. up grade^0.75
		βX = m.βU * gU
	case grade < 0:
		gD = -grade * grade // down grade^2
		βX = m.βD * gD
	}
	switch {
	case wind == 0:
	case wind < 0:
		βX += wind * (m.βT + m.βUT*gU + m.βDT*gD)
	case wind > 0:
		βX += wind * (m.βH + m.βUH*gU + m.βDH*gD)
	}
	if βX > m.logPowerUpRatio {
		return m.powerUpRatio
	}
	if βX < m.logPowerMinRatio {
		return 0
	}
	if useMathExp {
		return math.Exp(βX)
	}
	return fastExp(βX)
}

func (m *Generator) initSimpleLinear() {

	gU := m.sysGradeUp
	m.βU = (m.powerUpRatio - 1) / gU

	gD := m.sysGradeDown
	m.βD = (m.powerDownRatio - 1) / gD

	wT := m.sysTailwind
	m.βT = (m.powerTailRatio - 1) / wT

	wH := m.sysHeadwind
	m.βH = (m.powerHeadRatio - 1) / wH

	if m.powerDownHeadRatio > 0 {
		m.cDH = (m.powerDownHeadRatio - m.powerDownRatio - m.powerHeadRatio + 1)
		m.cDH /= (m.powerHeadRatio - 1)
	}
	if m.powerDownTailRatio > 0 {
		m.cDT = -(m.powerDownTailRatio - m.powerDownRatio - m.powerTailRatio + 1)
		m.cDT /= (m.powerTailRatio - 1)
	}
	m.βUH = -m.cUH * (m.powerHeadRatio - 1) / (gU * wH)
	m.βUT = -m.cUT * (m.powerTailRatio - 1) / (gU * wT)
	m.βDT = -m.cDT * (m.powerTailRatio - 1) / (gD * wT)
	m.βDH = +m.cDH * (m.powerHeadRatio - 1) / (gD * wH)
}

func (m *Generator) simpleLinear(grade, wind float64) (ratio float64) {

	var gU, gD, βX float64
	switch {
	case grade > 0:
		gU = grade
		βX = m.βU * gU
	case grade < 0:
		gD = grade
		βX = m.βD * gD
	}
	switch {
	case wind < 0:
		βX += wind * (m.βT + m.βUT*gU + m.βDT*gD)
	case wind > 0:
		βX += wind * (m.βH + m.βUH*gU + m.βDH*gD)
	}
	ratio = 1 + βX
	if ratio > m.powerUpRatio {
		return m.powerUpRatio
	}
	if ratio < 0 {
		return 0
	}
	return
}

func (m *Generator) initFullExponential() {

	fU := pow(m.sysGradeUp, m.expUphill)
	m.βU = ln(m.powerUpRatio) / fU

	fD := -pow(-m.sysGradeDown, m.expDownhill)
	m.βD = ln(m.powerDownRatio) / fD

	fT := -pow(-m.sysTailwind, m.expTailwind)
	m.βT = ln(m.powerTailRatio) / fT

	fH := pow(m.sysHeadwind, m.expHeadwind)
	m.βH = ln(m.powerHeadRatio) / fH

	if m.powerDownHeadRatio > 0 {
		m.cDH = m.powerDownHeadRatio / m.powerDownRatio
	}
	if m.powerDownTailRatio > 0 {
		m.cDT = m.powerDownTailRatio / m.powerDownRatio
	}

	m.βUT = (ln(m.cUT) - ln(m.powerTailRatio)) / (fU * fT)
	m.βUH = (ln(m.cUH) - ln(m.powerHeadRatio)) / (fU * fH)
	m.βDT = (ln(m.cDT) - ln(m.powerTailRatio)) / (fT * fD)
	m.βDH = (ln(m.cDH) - ln(m.powerHeadRatio)) / (fH * fD)
}

func (m *Generator) fullExponential(grade, wind float64) float64 {

	var gU, gD, βX float64
	switch {
	case grade > 0:
		gU = pow(grade, m.expUphill)
		βX = m.βU * gU
	case grade < 0:
		gD = -pow(-grade, m.expDownhill)
		βX = m.βD * gD
	}
	switch {
	case wind < 0:
		wind = -pow(-wind, m.expTailwind)
		βX += wind * (m.βT + m.βUT*gU + m.βDT*gD)
	case wind > 0:
		wind = pow(wind, m.expHeadwind)
		βX += wind * (m.βH + m.βUH*gU + m.βDH*gD)
	}
	if βX > m.logPowerUpRatio {
		return m.powerUpRatio
	}
	if βX < m.logPowerMinRatio {
		return 0
	}
	if useMathExp {
		return math.Exp(βX)
	}
	return fastExp(βX)
}

func (m *Generator) initFullLinear() {

	fG := pow(m.sysGradeUp, m.expUphill)
	m.βU = (m.powerUpRatio - 1) / fG

	fD := -pow(-m.sysGradeDown, m.expDownhill)
	m.βD = (m.powerDownRatio - 1) / fD

	fT := -pow(-m.sysTailwind, m.expTailwind)
	m.βT = (m.powerTailRatio - 1) / fT

	fH := pow(m.sysHeadwind, m.expHeadwind)
	m.βH = (m.powerHeadRatio - 1) / fH

	if m.cDH == 0 {
		m.cDH = (m.powerDownHeadRatio - m.powerDownRatio - m.powerHeadRatio + 1)
		m.cDH /= (m.powerHeadRatio - 1)
	}
	if m.cDT == 0 {
		m.cDT = -(m.powerDownTailRatio - m.powerDownRatio - m.powerTailRatio + 1)
		m.cDT /= (m.powerTailRatio - 1)
	}

	m.βUH = m.cUH * (1 - m.powerHeadRatio) / (fG * fH)
	m.βUT = m.cUT * (1 - m.powerTailRatio) / (fG * fT)
	m.βDT = m.cDT * (1 - m.powerTailRatio) / (fD * fT)
	m.βDH = m.cDH * (1 - m.powerHeadRatio) / (fD * fH)
}

func (m *Generator) fullLinear(grade, wind float64) float64 {

	gU, gD := 0.0, 0.0
	if grade > 0 {
		gU = pow(grade, m.expUphill)
	} else {
		gD = -pow(-grade, m.expDownhill)
	}
	ratio := 1.0 + m.βU*gU + m.βD*gD

	if wind == 0 {
		return max(0, ratio)
	}
	if wind < 0 {
		wind = -pow(-wind, m.expTailwind)
		return max(0, ratio+wind*(m.βT+m.βUT*gU+m.βDT*gD))
	}
	wind = pow(wind, m.expHeadwind)
	return max(0, ratio+wind*(m.βH+m.βUH*gU+m.βDH*gD))
}

// R-estimation of coefficients of fastExp
// minimaxApprox(exp, -5, 2, degree=c(3,3))
// $ObservedAbsError 0.0002340462

// fastExp returns approx. exp(x) for x in [-5, 2]. Max error ~ 0.00025.
// This is ~8 x faster than math.Exp.
func fastExp(x float64) (z float64) {
	const (
		a1 = +0.465807347
		a2 = +0.081577544
		a3 = +0.005276719
		b1 = -0.5335895
		b2 = +0.11496513
		b3 = -0.01017503
	)
	if useFMA {
		return math.FMA(x, math.FMA(x, math.FMA(x, a3, a2), a1), 1) /
			math.FMA(x, math.FMA(x, math.FMA(x, b3, b2), b1), 1)
	}
	return (1 + x*a1 + x*x*(a2+x*a3)) / (1 + x*b1 + x*x*(b2+x*b3))
}

// fn = function(x) x^0.75
// minimaxApprox(fn, 0.005, 0.1, c(3))
// 0.006923987   2.563897835 -14.621961493 61.316361623
// $ObservedAbsError 0.0005825766

// pow34 returns ~x^(3/4) for x in [0, 0.1]. Max error 0.00058
func pow34(x float64) float64 {
	const (
		a0 = +0.006923987
		a1 = +2.563897835
		a2 = -14.621961493
		a3 = +61.316361623
	)
	if x < 0.005 {
		return math.Sqrt(x * math.Sqrt(x))
	}
	return a0 + x*(a1+x*(a2+x*a3))
}

// fU34 returns ~x^(3/4) for x in [0, 0.1].
func fU34(x float64) float64 {
	const (
		// a0 = +0.002099753
		a1 = +3.044942978
		a2 = -25.397962088
		a3 = +127.313264131
	)
	return x * (a1 + x*(a2+x*a3))
}

// pow implements function math.Pow(x,y) = x^y faster for common y's in this system
func pow(x, y float64) float64 {
	switch y {
	case 1:
		return x
	case 2:
		return x * x
	case 0.75:
		return math.Sqrt(math.Sqrt(x) * x)
	case 1.5:
		return math.Sqrt(x) * x
	case 0.5:
		return math.Sqrt(x)
	case 2.0 / 3:
		x = math.Cbrt(x)
		return x * x
	}
	return math.Pow(x, y)
}

// PowerModelType --------
func (m *Generator) PowerModelType(i int) {
	m.powerModelType = i
}

// SysGradeUp -----------
func (m *Generator) SysGradeUp(x float64) {
	m.sysGradeUp = x
}

// SysGradeDown ----------
func (m *Generator) SysGradeDown(x float64) {
	m.sysGradeDown = x
}

// SysTailwind ---------
func (m *Generator) SysTailwind(x float64) {
	m.sysTailwind = x
}

// SysHeadwind ---------
func (m *Generator) SysHeadwind(x float64) {
	m.sysHeadwind = x
}

// ExpDownhill ----------
func (m *Generator) ExpDownhill(x float64) {
	m.expDownhill = x
}

// ExpHeadwind ------------
func (m *Generator) ExpHeadwind(x float64) {
	m.expHeadwind = x
}

// ExpTailwind -----------
func (m *Generator) ExpTailwind(x float64) {
	m.expTailwind = x
}

// ExpUphill ----------
func (m *Generator) ExpUphill(x float64) {
	m.expUphill = x
}

// PowerDownRatio -------
func (m *Generator) PowerDownRatio(x float64) {
	m.powerDownRatio = x
}

// PowerTailRatio --------
func (m *Generator) PowerTailRatio(x float64) {
	m.powerTailRatio = x
}

// PowerHeadRatio --------
func (m *Generator) PowerHeadRatio(x float64) {
	m.powerHeadRatio = x
}

// PowerDownHeadRatio --------
func (m *Generator) PowerDownHeadRatio(x float64) {
	m.powerDownHeadRatio = x
}

// PowerDownTailRatio --------
func (m *Generator) PowerDownTailRatio(x float64) {
	m.powerDownTailRatio = x
}

// PowerUpRatio ----------
func (m *Generator) PowerUpRatio(x float64) {
	m.powerUpRatio = x
}
func (m *Generator) PowerMinpRatio(x float64) {
	m.powerMinRatio = x
}

// CDT ------
func (m *Generator) CDT(x float64) {
	m.cDT = x
}

// CUT --------
func (m *Generator) CUT(x float64) {
	m.cUT = x
}

// CDH ----------
func (m *Generator) CDH(x float64) {
	m.cDH = x
}

// CUH -------
func (m *Generator) CUH(x float64) {
	m.cUH = x
}
