package route

import (
	"bikeride/param"
	"errors"
)

type par *param.Parameters

const (
	testing     = false
	distTOL     = 0.05  // m
	powerTOL    = 1.0   // W
	deltaVelTOL = 1e-18 // m/s

)

const (
	mh2ms   = 1.0 / 3600
	s2h     = 1.0 / 3600
	kmh2ms  = 1.0 / 3.6
	ms2kmh  = 3.6
	m2km    = 1.0 / 1000
	sec2min = 1.0 / 60
	j2Wh    = 1.0 / 3600
	kj2wh   = 1.0 / 3.6
)

/*
https://fineli.fi/fineli/en/elintarvikkeet/11049?portionUnit=KPL_M&portionSize=1
Banana, without skin: 1 medium sized piece 125 g = 458 kJ
Lard, Frying Fat 3591 kJ / 100 g
Lard is a semi-solid white fat product obtained by rendering
the fatty tissue of a pig. It is distinguished from tallow,
a similar product derived from fat of cattle or sheep.
Tallow, Beef Fat: 3,684 kJ / 100 g
*/
const (
	humanEfficiency = 0.24
	banana2Wh       = 458.0 * kj2wh
	lard2Wh         = 3591.0 * kj2wh
	j2banana        = j2Wh / (banana2Wh * humanEfficiency)
	j2lard          = 100.0 * j2Wh / (lard2Wh * humanEfficiency)
	j2kcal          = (1.0 / 4184) / humanEfficiency
)
const (
	freewheeling    = 0
	acceleration    = 1
	deceleration    = 2
	braking         = 3
	ridingConstVel  = 4
	brakingConstVel = 5
	oddCase         = 7
	noAcceleration  = 9
)

func errf(s string) error { return errors.New(s) }

func min(x, y float64) float64 {
	if x < y {
		return x
	}
	return y
}
func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}
func (o *Route) Segments() int { return o.segments }

type route []segment

type filter struct {
	ipoBackStepRounds int
	backsteps         int
	initRelgrade      float64
	minRelGrade       float64
	levelFactor       float64
	levelMax          float64
	levelMin          float64
	ipoDist           float64
	ipoSumDist        float64
	smoothingWeight   float64
	maxAcceptedGrade  float64
	ipolations        int
	levelations       int
	eleLeveled        float64
	distFilterGrade   float64
	minSegDist        float64
}

type Route struct {
	route        route
	filter       filter
	trkpErrors   int
	trkpRejected int
	segments     int

	distance   float64
	distDirect float64
	distMean   float64
	distMedian float64

	routeCourse float64
	windCourse  float64
	windSin     float64
	windCos     float64
	windSpeed   float64
	metersLon   float64
	metersLat   float64

	limitTurnSpeeds bool

	eleUp      float64
	eleDown    float64
	eleUpGPX   float64
	eleDownGPX float64
	eleMax     float64
	eleMin     float64
	eleMissing int
	vMaxToNext int

	jouleRider float64
	time       float64

	//public for package main
	EleMean     float64
	LatMean     float64
	Gravity     float64
	Temperature float64
	Rho         float64
}

type Results struct {
	WindCourse    float64
	WindSpeed     float64
	RouteCourse   float64
	BaseElevation float64
	AirPressure   float64
	MeanElevation float64
	Temperature   float64
	Rho           float64
	RhoBase       float64
	Segments      int
	TrkpErrors    int
	TrkpRejected  int
	DistTotal     float64
	DistDirect    float64
	DistBrake     float64
	DistFreewheel float64
	DistUphill    float64
	DistDownhill  float64
	DistFlat      float64
	DistMedian    float64
	DistMean      float64
	DistMax       float64
	DistMin       float64

	EleUp        float64
	EleUpKinetic float64
	EleDown      float64
	EleUpGPX     float64
	EleDownGPX   float64
	EleMax       float64
	EleMin       float64
	EleLevelled  float64
	EleMean      float64
	EleMissing   int

	LatMean float64
	Gravity float64

	Filtered     float64
	Filterable   float64
	FilteredPros float64
	Ipolations   int
	Levelations  int
	FilterRounds int
	MinGrade     float64
	MaxGrade     float64
	RelGradeSD   float64

	Time              float64
	TimeRider         float64
	TimeBraking       float64
	TimeFreewheel     float64
	TimeUHBreaks      float64
	TimeFullPower     float64
	TimeOverFlatPower float64
	TimeTargetSpeeds  float64
	TimeDownhill      float64

	VelAvg             float64
	VelMax             float64
	VelMin             float64
	VelDownhill        float64
	VelDownVert        float64
	VerticalDownEle    float64
	DownhillMaxSpeed   float64
	MaxGradeUp         float64
	DownhillPowerSpeed float64

	JriderTotal float64
	FoodRider   float64
	BananaRider float64
	FatRider    float64

	JfromTargetPower float64
	JriderFullPower  float64
	JriderGravUp     float64
	JriderDrag       float64
	JriderRoll       float64
	JriderAcce       float64
	JlossDT          float64
	PowerRiderAvg    float64
	EnergySumRider   float64

	JkineticDece     float64
	JkineticAcce     float64
	JdragRider       float64
	JdragBrake       float64
	JdragFreewheel   float64
	JdragResistance  float64
	JdragPush        float64
	Jroll            float64
	JgravUp          float64
	JgravDown        float64
	Jbraking         float64
	Jsink            float64
	EnergySumTotal   float64
	SegEnergyMeanAbs float64
	SegEnergyMean    float64

	SolverRoundsAvg   float64
	VelErrorMean      float64
	VelErrorAbsMean   float64
	VelErrorPos       float64
	VelErrorMax       float64
	MaxIter           int
	SolverCalls       int
	FreewheelCalls    int
	PowerFromVelCalls int
	CalcSteps         int
	CalcSegs          int
	CalcStepsAvg      float64
	SingleStepPros    float64
}

type segment struct {
	segnum  int
	lon     float64
	lat     float64
	ele     float64
	eleGPX  float64
	grade   float64
	dist    float64
	distHor float64
	course  float64
	radius  float64
	wind    float64

	ipolable   bool
	ipolations int

	powerTarget float64
	vTarget     float64
	vEntry      float64
	vExit       float64
	vMax        float64
	vEntryMax   float64

	jouleRider      float64
	jouleDragRider  float64
	jouleDeceRider  float64
	jouleGrav       float64
	jouleRoll       float64
	jouleDrag       float64
	jouleDragFreewh float64
	jouleKinetic    float64
	jouleBraking    float64
	jouleDragBrake  float64
	jouleSink       float64

	powerRider   float64
	powerBraking float64

	distKinetic   float64
	distLeft      float64
	distBraking   float64
	distFreewheel float64

	time          float64
	timeRider     float64
	timeBraking   float64
	timeFreewheel float64
	timeBreak     float64

	calcSteps int
	calcPath  int
}
