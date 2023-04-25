package param

const (
	kmh2ms  = 1.0 / 3.6
	ms2kmh  = 3.6
	min2sec = 60.0
	sec2min = 1.0 / 60.0
)

type logger interface {
	Msg(level int, v ...interface{}) error
	Err(v ...interface{}) error
	Errorf(format string, v ...interface{}) error
	Errors() int
	SetPrefix(string)
}

type Parameters struct {
	Filter      filter
	Weight      weight      `json:"weight (kg)"`
	UphillBreak uphillBreak `json:"uphillBreaks"`
	Powermodel  powermodel
	Ride        ride

	bike
	environment
	calculation
	filesEtc
}

type filter struct {
	IpoBackStepRounds int     `json:"interpolateBackstepRounds"`
	InitialRelGrade   float64 `json:"initialRelativeGrade"`
	MinRelGrade       float64 `json:"minRelativeGrade"`
	IpoSumDist        float64 `json:"interpolateSumDist"`
	IpoDist           float64 `json:"interpolateDist"`
	Backsteps         int     `json:"interpolateBacksteps"`
	LevelFactor       float64
	LevelMax          float64
	LevelMin          float64
	WeightedAvgWeight float64 `json:"smoothingWeight"`
	MaxAcceptedGrade  float64 `json:"maxAcceptedGrade"`
	MinSegDist        float64 `json:"minSegmentDistance"`
	DistFilterGrade   float64 `json:"distInterpolationGrade"`
}

// DiffCalc	= 1 stepping delta velocity
// DiffCalc	= 2 stepping delta distance
// DiffCalc	= 3 stepping delta time

type calculation struct {
	DeltaVel  float64
	DeltaTime float64
	DiffCalc  int `json:"acceStepMode"`
	VelSolver int

	NRtol           float64 `json:"velSolverTol"`
	Bracket         float64 `json:"velSolverBracket"`
	ReportTech      bool
	VelErrors       bool `json:"reportVelErrors"`
	UseVelTable     bool
	PowerIn         float64 // (100 - DrivetrainLoss) / 100
	PowerOut        float64 // 1 / PowerIn
	VerticalUpGrade float64
	MinPedaledGrade float64
}

type filesEtc struct {
	RideJSON        string
	ConfigJSON      string
	GPXfile         string
	RouteName       string
	GPXdir          string
	ResultDir       string
	RouteCSV        bool
	ResultTXT       bool
	ResultJSON      bool
	Logfile         string
	ParamOutJSON    bool
	UseCR           bool
	CSVuseTab       bool
	GPXuseXMLparser bool `json:"GPXuseXMLparser"`
	GPXignoreErrors bool `json:"GPXignoreErrors"`
	Display         bool
	LogMode         int
	LogLevel        int
	CheckParams     bool
}
type ride struct {
	MaxSpeed           float64 `json:"maxSpeed (km/h)"`
	MinSpeed           float64 `json:"minSpeed (km/h)"`
	MinLimitedSpeed    float64 `json:"minLimitedSpeed (km/h)"`
	LimitDownSpeeds    bool
	VerticalDownSpeed  float64 `json:"verticalDownSpeed (m/h)"`
	SteepDownhillGrade float64
	SpeedLimitGrade    float64 `json:"speedLimitGrade"`
	BrakingDist        float64 `json:"brakingDistance (m)"`
	Cbf                float64 `json:"brakeRoadFriction"`
	LimitTurnSpeeds    bool
	Ccf                float64 `json:"turnFriction"`
	LimitEntrySpeeds   bool
	PowerAcce          float64 `json:"accelerationPower (%)"`
	PowerAcceMin       float64 `json:"minAccelerationPower (w)"`
	PowerDece          float64 `json:"decelerationPower (%)"`
	VelDeceLim         float64 `json:"deceFreewheelCutoff"`
	KeepEntrySpeed     bool    `json:"keepEntrySpeed"`
}

type weight struct {
	Rider   float64
	Bike    float64
	Luggage float64
	Total   float64
	Wheels  float64
}

type uphillBreak struct {
	PowerLimit    float64 `json:"powerLimit (%)"`
	ClimbDuration float64 `json:"climbDuration (min)"`
	BreakDuration float64 `json:"breakDuration (min)"`
}

type powermodel struct {
	PowermodelType int `json:"powerModel"`

	FlatSpeed float64 `json:"flatGroundSpeed (km/h)"`
	FlatPower float64 `json:"flatGroundPower (w)"`

	UphillPowerGrade float64
	UphillPowerSpeed float64 `json:"uphillPowerSpeed (km/h)"`
	UphillPower      float64 `json:"uphillPower (w)"`
	VerticalUpSpeed  float64 `json:"verticalUpSpeed (m/h)"`

	DownhillPowerGrade float64
	DownhillPower      float64 `json:"downhillPower (%)"`
	MaxPedaledSpeed    float64 `json:"maxPedaledSpeed (km/h)"`

	SysTailwind           float64 `json:"sysTailwind (m/s)"`
	SysHeadwind           float64 `json:"sysHeadwind (m/s)"`
	TailWindPower         float64 `json:"tailWindPower (%)"`
	HeadWindPower         float64 `json:"headWindPower (%)"`
	ExpDownhill           float64
	ExpUphill             float64
	ExpHeadwind           float64
	ExpTailwind           float64
	DownhillTailwindPower float64 `json:"downhillTailwindPower (%)"`
	DownhillHeadwindPower float64 `json:"downhillHeadwindPower (%)"`
	CDT                   float64
	CDH                   float64
	CUT                   float64
	CUH                   float64
}

type bike struct {
	CdA            float64 `json:"airDragCoef CdA"`
	Cd             float64
	FrontalArea    float64
	DrivetrainLoss float64 `json:"drivetrainLoss (%)"`
	Crr            float64 `json:"rollingResistance"`
}

type environment struct {
	WindCourse    float64 `json:"windCourse (deg)"`
	WindSpeed     float64 `json:"windSpeed (m/s)"`
	BaseElevation float64
	Temperature   float64 `json:"temperature (C)"`
	AirDensity    float64 `json:"airDensity (rho)"`
	AirPressure   float64 `json:"airPressure (hPa)"`
	Gravity       float64
}
