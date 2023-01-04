package power

import "math"

const (
	simpleExponential = 1
	simpleLinear      = 2
	fullExponential   = 3
	fullLinear        = 4
)

var log = math.Log

var ratioModel func(*Generator, float64, float64) float64

// Generator ----
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

// RatioGenerator tekee alustetun tehomallin
func RatioGenerator() *Generator {
	return &Generator{
		cDT: 1,
		cUT: 1,
		cDH: 1,
		cUH: 1,

		sysGradeUp:   0.06,
		sysGradeDown: -0.025,
		sysTailwind:  -5,
		sysHeadwind:  5,

		expDownhill: 2,
		expUphill:   0.75,
		expHeadwind: 1,
		expTailwind: 1,

		powerDownRatio:     0.2,
		powerTailRatio:     0.85,
		powerHeadRatio:     1.35,
		powerUpRatio:       1.75,
		powerDownHeadRatio: 1,
		powerDownTailRatio: 0.05,
		powerModelType:     1,
		powerMinRatio:      0.005,
	}
}

// Setup --
func (m *Generator) Setup() {

	if m.powerUpRatio < 1 {
		m.powerUpRatio = 1
	}
	m.logPowerUpRatio = log(m.powerUpRatio)
	m.logPowerMinRatio = log(m.powerMinRatio)

	switch m.powerModelType {
	default:
		initSimpleExponential(m)
		ratioModel = (*Generator).simpleExponential

	case simpleLinear:
		initSimpleLinear(m)
		ratioModel = (*Generator).simpleLinear

	case fullExponential:
		initFullExponential(m)
		ratioModel = (*Generator).fullExponential

	case fullLinear:
		initFullLinear(m)
		ratioModel = (*Generator).fullLinear
	}
}

// Ratio --
func (m *Generator) Ratio(grade, wind float64) float64 {

	if grade >= m.sysGradeUp {
		return m.powerUpRatio
	}
	wind = m.cutWind(grade, wind)

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
	return m.sysHeadwind + (wind-m.sysHeadwind)*(grade-m.sysGradeDown)/grade*0.25
}

// Uphill grade function fU = grade^0.75 gives quite linear power increase.
func initSimpleExponential(m *Generator) {

	fU := math.Sqrt(m.sysGradeUp * math.Sqrt(m.sysGradeUp)) // grade^0.75
	m.βU = log(m.powerUpRatio) / fU

	fD := -(m.sysGradeDown * m.sysGradeDown)
	m.βD = log(m.powerDownRatio) / fD

	fT := m.sysTailwind
	m.βT = log(m.powerTailRatio) / fT

	fH := m.sysHeadwind
	m.βH = log(m.powerHeadRatio) / fH

	if m.powerDownHeadRatio > 0 {
		m.cDH = m.powerDownHeadRatio / m.powerDownRatio
	}
	if m.powerDownTailRatio > 0 {
		m.cDT = m.powerDownTailRatio / m.powerDownRatio
	}
	m.βUT = (log(m.cUT) - log(m.powerTailRatio)) / (fU * fT)
	m.βUH = (log(m.cUH) - log(m.powerHeadRatio)) / (fU * fH)
	m.βDT = (log(m.cDT) - log(m.powerTailRatio)) / (fD * fT)
	m.βDH = (log(m.cDH) - log(m.powerHeadRatio)) / (fD * fH)
}

func (m *Generator) simpleExponential(grade float64, wind float64) float64 {

	var gU, gD, βX float64
	if grade > 0 {
		gU = math.Sqrt(grade * math.Sqrt(grade)) //up grade^0.75
		βX = m.βU * gU
	} else {
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
	return fastExp(βX)
}

func initSimpleLinear(m *Generator) {

	gU := m.sysGradeUp
	m.βU = (m.powerUpRatio - 1) / gU

	gD := m.sysGradeDown
	// gD := -m.sysGradeDown * m.sysGradeDown
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

	gU, gD := grade, 0.0
	if grade < 0 {
		gU, gD = 0.0, grade
	}
	βX := m.βU*gU + m.βD*gD

	switch {
	case wind == 0:
	case wind < 0:
		βX += wind * (m.βT + m.βUT*gU + m.βDT*gD)
	case wind > 0:
		βX += wind * (m.βH + m.βUH*gU + m.βDH*gD)
	}
	ratio = 1.0 + βX
	if ratio > m.powerUpRatio {
		return m.powerUpRatio
	}
	if ratio < 0 {
		return 0
	}
	return
}

func initFullExponential(m *Generator) {

	fU := pow(m.sysGradeUp, m.expUphill)
	m.βU = log(m.powerUpRatio) / fU

	fD := -pow(-m.sysGradeDown, m.expDownhill)
	m.βD = log(m.powerDownRatio) / fD

	fT := -pow(-m.sysTailwind, m.expTailwind)
	m.βT = log(m.powerTailRatio) / fT

	fH := pow(m.sysHeadwind, m.expHeadwind)
	m.βH = log(m.powerHeadRatio) / fH

	if m.powerDownHeadRatio > 0 {
		m.cDH = m.powerDownHeadRatio / m.powerDownRatio
	}
	if m.powerDownTailRatio > 0 {
		m.cDT = m.powerDownTailRatio / m.powerDownRatio
	}

	m.βUT = (log(m.cUT) - log(m.powerTailRatio)) / (fU * fT)
	m.βUH = (log(m.cUH) - log(m.powerHeadRatio)) / (fU * fH)
	m.βDT = (log(m.cDT) - log(m.powerTailRatio)) / (fT * fD)
	m.βDH = (log(m.cDH) - log(m.powerHeadRatio)) / (fH * fD)
}

func (m *Generator) fullExponential(grade, wind float64) float64 {

	gU, gD := 0.0, 0.0
	if grade > 0 {
		gU = pow(grade, m.expUphill)
	} else {
		gD = -pow(-grade, m.expDownhill)
	}
	βX := m.βU*gU + m.βD*gD
	switch {
	case wind == 0:
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
	return math.Exp(βX)
}

func initFullLinear(m *Generator) {

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

func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}

// e^x ~ (1 + x/4096)^4096. x < 2 => error < 0.004
func fastExp(x float64) float64 {
	x = 1 + x*(1.0/4096.0)
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	x *= x
	return x
}

// pow implements function math.Pow(x,y) = x^y  faster for common y's in this system
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
	case 2.0 / 3.0:
		return math.Cbrt(x * x)
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
