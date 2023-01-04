package param

import (
	"encoding/json"
	"os"
)

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

// Parameters on --
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
	Rounds          int
	InitialRelGrade float64 `json:"initialRelativeGrade"`
	MinRelGrade     float64 `json:"minRelativeGrade"`
	IpoSumDist      float64 `json:"interpolateSumDist"`
	IpoDist         float64 `json:"interpolateDist"`
	Backsteps       int
	LevelFactor     float64
	LevelMax        float64
	LevelMin        float64
	MovingAvgWeight float64 `json:"movingAverageWeight"`
	MinSegDist      float64 `json:"minSegmentDist"`
	MaxFilteredEle  float64 `json:"maxFilteredElevation"`
	MaxIpolations   int     `json:"maxInterpolations"`
}

// DiffCalc	= 1 stepping delta velocity
// DiffCalc	= 2 stepping delta distance
// DiffCalc	= 3 stepping delta time

type calculation struct {
	DeltaVel  float64
	DeltaTime float64
	DiffCalc  int `json:"diffCalc"`
	VelSolver int

	NRtol             float64 `json:"velTolerance"`
	Bracket           float64 `json:"velBracket"`
	ReportTech        bool
	VelErrors         bool `json:"calcVelError"`
	UseVelTable       bool
	SingleStepBraking bool
	PowerIn           float64
	PowerOut          float64
	VerticalUpGrade   float64
	MinPedaledGrade   float64
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
	CheckRideSetup  bool
}
type ride struct {
	MaxSpeed           float64 `json:"maxSpeed (km/h)"`
	MinSpeed           float64 `json:"minSpeed (km/h)"`
	MinLimitedSpeed    float64 `json:"minLimitedSpeed (km/h)"`
	LimitDownSpeeds    bool
	VerticalDownSpeed  float64 `json:"verticalDownSpeed (m/h)"`
	SteepDownhillGrade float64
	VelLimitGrade      float64
	BrakingDist        float64 `json:"brakingDist (m)"`
	Cbf                float64 `json:"brakingFriction"`
	LimitTurnSpeeds    bool
	Ccf                float64 `json:"turnFriction"`
	LimitEntrySpeeds   bool
	PowerAcce          float64 `json:"accelerationPower (%)"`
	PowerAcceMin       float64 `json:"minAccelerationPower (%)"`
	PowerDece          float64 `json:"decelerationPower (%)"`
	VelDeceLim         float64 `json:"deceFreeWheelCutoff"`
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

// New --
func New(args []string, l logger) (*Parameters, error) {

	p := &Parameters{}
	p.setSystemDefaultValues()

	cfgfile := commandLineArg("-cfg", args)
	if cfgfile == "" {
		cfgfile = "config.json"
	}
	bytes, err := os.ReadFile(cfgfile)
	if err != nil {
		return p, l.Errorf("Config file: %v", err)
	}
	if err = json.Unmarshal(bytes, &p); err != nil {
		return p, l.Errorf("File "+cfgfile+" - %v", err)
	}
	p.ConfigJSON = cfgfile

	p.RideJSON = args[1]
	bytes, err = os.ReadFile(p.RideJSON)
	if err != nil {
		return p, l.Errorf("Ride parameter file: %v", err)
	}
	if err = json.Unmarshal(bytes, &p); err != nil {
		return p, l.Errorf("File "+p.RideJSON+" - %v", err)
	}
	gpxfile := commandLineArg("-gpx", args)
	if gpxfile != "" {
		if p.RouteName == "" {
			p.RouteName = routeNameFromFile(gpxfile)
		}
		p.GPXfile = gpxfile
	}
	if p.ResultDir == "" {
		return p, nil
	}
	if _, err := os.Stat(p.ResultDir); os.IsNotExist(err) {
		if err := os.Mkdir(p.ResultDir, os.ModeDir); err != nil {
			//TODO this doesn't always work
			return p, l.Errorf("Results dir: %v", err)
		}
	}
	return p, nil
}

func commandLineArg(arg string, args []string) string {
	for i := 1; i < len(args)-1; i++ {
		if args[i] == arg {
			return args[i+1]
		}
	}
	return ""
}

func routeNameFromFile(filename string) string {
	//take bytes before possible dot
	b := []byte(filename)
	i := 0
	for i < len(b) && b[i] != '.' {
		i++
	}
	return string(b[0:i])
}

func (p *Parameters) setSystemDefaultValues() {

	p.GPXfile = ""
	p.GPXdir = "./gpx/"
	p.ResultDir = "./results/"
	p.RouteCSV = false
	p.ResultTXT = true
	p.ResultJSON = false
	p.ParamOutJSON = false
	p.Logfile = "log.txt"
	p.LogMode = 0
	p.LogLevel = 1
	p.UseCR = true
	p.CSVuseTab = false
	p.Display = true
	p.CheckParams = true
	p.ReportTech = false

	p.Filter.Rounds = 2
	p.Filter.IpoDist = 0
	p.Filter.IpoSumDist = 0
	p.Filter.InitialRelGrade = 0
	p.Filter.MinRelGrade = 0
	p.Filter.Backsteps = 5
	p.Filter.LevelFactor = 0.5
	p.Filter.LevelMax = 4
	p.Filter.LevelMin = 1
	p.Filter.MaxFilteredEle = 12

	Q := &p.Powermodel
	Q.PowermodelType = 1
	Q.TailWindPower = 85
	Q.HeadWindPower = 125
	Q.SysTailwind = -5
	Q.SysHeadwind = 5
	Q.ExpDownhill = 2
	Q.ExpUphill = 0.75
	Q.ExpHeadwind = 1
	Q.ExpTailwind = 1
	Q.DownhillTailwindPower = 5
	Q.DownhillHeadwindPower = 100
	Q.CUT = 1
	Q.CUH = 1
	Q.CDT = 1
	Q.CDH = 1

	R := &p.Ride
	R.MaxSpeed = 100
	R.LimitTurnSpeeds = false
	R.LimitDownSpeeds = true
	R.LimitEntrySpeeds = false
	R.MinLimitedSpeed = 7.2
	R.MinSpeed = 3
	R.SteepDownhillGrade = -10
	R.PowerAcce = 120
	R.PowerAcceMin = 50
	R.PowerDece = 80
	R.VelDeceLim = 0.5
	// R.HeavyBrakingPower = -500
	R.VelLimitGrade = -3

	p.DeltaVel = 0.5
	p.DeltaTime = 0.75
	p.DiffCalc = 1
	p.VelSolver = 1
	p.NRtol = 0.05
	p.Bracket = 1.5
	p.VelErrors = false
	p.UseVelTable = false
	p.VerticalUpGrade = 7
	p.SingleStepBraking = true

}

type attributes struct {
	min      float64
	max      float64
	unit     string
	notGiven float64
}

const mustGiven = 1e10

type attributesMap map[string]attributes

func (m attributesMap) put(key string, min, max float64, unit string, notGiven float64) {
	m[key] = attributes{min, max, unit, notGiven}
}
func (m attributesMap) check(val float64, key string, l logger) {
	a, found := m[key]
	if !found {
		l.Err("No attributes found for " + key)
		return
	}
	if a.min <= val && val <= a.max {
		return
	}
	if val == a.notGiven && a.notGiven != mustGiven {
		return
	}
	if a.notGiven == mustGiven {
		l.Err("Out of range "+key+" =", val, "(range", a.min, "-", a.max, a.unit+")")
		return
	}
	l.Err("Out of range "+key+" =", val, "(range", a.min, "-", a.max, a.unit+" and not given =", a.notGiven, ")")
}

func paramRanges() attributesMap {
	m := make(attributesMap, 100)
	m.put("weight.total", 40, 10000, "kg", 0)
	m.put("windCourse", 0, 360, "degrees", -1)
	m.put("windSpeed", -10, 10, "m/s", mustGiven)
	m.put("airDensity", 0.72, 1.5, "", -1)
	m.put("gravity", 9.780, 9.833, "", -1)
	m.put("temperature", -25, 45, "deg C", mustGiven)
	m.put("airPressure", 950, 1085, "hPascals", -1)
	m.put("baseElevation", 0, 5000, "m", mustGiven)
	m.put("minSegmentDist", 0, 100, "m", mustGiven)

	m.put("flatGroundSpeed", 10, 60, "km/h", -1)
	m.put("flatGroundPower", 25, 500, "w", -1)

	m.put("verticalUpSpeed", 100, 2500, "m/h", -1)
	m.put("uphillPower", 75, 750, "w", -1)
	m.put("uphillPowerSpeed", 3, 25, "km/h", -1)
	m.put("uphillPowerGrade", 2, 10, "%", -1)

	m.put("downhillPower", 10, 25, "%", mustGiven)
	m.put("downhillPowerGrade", -8, -1.5, "%", -1)
	m.put("maxPedalledSpeed", 15, 70, "km/h", mustGiven)
	m.put("verticalDownSpeed", 500, 6000, "m/h", -1)
	m.put("brakingDist", 5, 100, "m", -1)

	m.put("tailWindPower", 50, 100, "%", mustGiven)
	m.put("sysTailWind", -8, -2, "m/s", mustGiven)
	m.put("sysHeadwind", 2, 8, "m/s", mustGiven)
	m.put("headWindPower", 100, 150, "%", mustGiven)
	m.put("downhillHeadwindPower", 50, 125, "%", mustGiven)
	m.put("downhillTailwindPower", 1, 20, "%", mustGiven)

	m.put("decelerationPower", 25, 95, "%", mustGiven)
	m.put("accelerationPower", 105, 150, "%", mustGiven)
	m.put("minAccelerationPower", 25, 200, "%", mustGiven)
	m.put("deceFreeWheelCutoff", 0, 1, "", mustGiven)

	m.put("rollingResistanceCoef", 0.001, 0.02, "", mustGiven)
	m.put("brakingFrictionCoef", 0.1, 0.5, "", mustGiven)
	m.put("turnFrictionCoef", 0.05, 0.5, "", mustGiven)
	m.put("airDragCoef CdA", 0.01, 1.5, "", -1)
	m.put("drivetrainLoss", 1, 20, "%", mustGiven)

	m.put("maxSpeed", 20, 1000, "km/h", mustGiven)
	m.put("minSpeed", 1, 15, "km/h", -1)
	m.put("minLimitedSpeed", 5, 15, "km/h", mustGiven)
	m.put("uphillBreak.powerLimit", 75, 100, "%", 0)
	m.put("uphillBreak.breakDuration", 1, 20, "min", 0)
	m.put("uphillBreak.climbDuration", 5, 60, "min", 0)
	m.put("filter.levelFactor", 0.1, 1, "", 0)
	m.put("filter.levelMax", 1, 30, "m", mustGiven)
	m.put("filter.levelMin", 0.25, 5, "m", mustGiven)
	return m
}

// Check ---
func (p *Parameters) Check(l logger) error {
	l.SetPrefix("param:      ")
	Q := &p.Powermodel

	if Q.FlatSpeed <= 0 && Q.FlatPower <= 0 {
		l.Err("flatGroundSpeed <= 0 and flatGroundPower <= 0")
	}
	if Q.VerticalUpSpeed <= 0 && Q.UphillPower <= 0 {
		l.Err("verticalUpSpeed <= 0 and uphillPower <= 0")
	}
	if Q.UphillPowerSpeed <= 0 && Q.UphillPowerGrade <= 0 {
		l.Err("uphillPowerSpeed <= 0 and uphillPowerGrade <= 0")
	}
	if Q.MaxPedaledSpeed <= 0 && Q.DownhillPowerGrade <= 0 {
		l.Err("maxPedalledSpeed <= 0 and downhillPowerGrade <= 0")
	}
	if p.CdA <= 0 && (Q.FlatSpeed <= 0 || Q.FlatPower <= 0) {
		l.Err("airDragCoef CdA <= 0 and (flatSpeed <= 0 or flatPower <= 0)")
	}
	U := &p.UphillBreak
	if U.ClimbDuration > 0 && U.BreakDuration > 0 && U.BreakDuration > U.ClimbDuration {
		l.Err("uphillBreakDuration  > uphillBreakTimeBracket")
	}
	if p.Weight.Total <= 0 {
		p.Weight.Total = p.Weight.Bike + p.Weight.Rider + p.Weight.Luggage
	}
	if p.CheckParams {
		m := paramRanges()
		checkParamRange(p, m, l)
	}
	if l.Errors() > 0 {
		return l.Errorf(" ")
	}
	return nil
}

func checkParamRange(p *Parameters, m attributesMap, l logger) {

	R := &p.Ride
	Q := &p.Powermodel

	m.check(p.Weight.Total, "weight.total", l)
	m.check(p.WindCourse, "windCourse", l)
	m.check(p.WindSpeed, "windSpeed", l)
	m.check(p.AirDensity, "airDensity", l)
	m.check(p.Gravity, "gravity", l)
	m.check(p.Temperature, "temperature", l)
	m.check(p.AirPressure, "airPressure", l)
	m.check(p.BaseElevation, "baseElevation", l)
	m.check(R.VerticalDownSpeed, "verticalDownSpeed", l)
	m.check(R.BrakingDist, "brakingDist", l)

	m.check(Q.FlatSpeed, "flatGroundSpeed", l)
	m.check(Q.FlatPower, "flatGroundPower", l)
	m.check(Q.VerticalUpSpeed, "verticalUpSpeed", l)
	m.check(Q.UphillPower, "uphillPower", l)
	m.check(Q.UphillPowerSpeed, "uphillPowerSpeed", l)
	m.check(Q.UphillPowerGrade, "uphillPowerGrade", l)
	m.check(Q.DownhillPowerGrade, "downhillPowerGrade", l)
	m.check(Q.MaxPedaledSpeed, "maxPedalledSpeed", l)
	m.check(Q.TailWindPower, "tailWindPower", l)
	m.check(Q.SysTailwind, "sysTailWind", l)
	m.check(Q.SysHeadwind, "sysHeadwind", l)
	m.check(Q.HeadWindPower, "headWindPower", l)
	m.check(Q.DownhillPower, "downhillPower", l)
	m.check(Q.DownhillHeadwindPower, "downhillHeadwindPower", l)
	m.check(Q.DownhillTailwindPower, "downhillTailwindPower", l)

	m.check(R.PowerDece, "decelerationPower", l)
	m.check(R.PowerAcce, "accelerationPower", l)
	m.check(R.PowerAcceMin, "minAccelerationPower", l)
	m.check(R.VelDeceLim, "deceFreeWheelCutoff", l)
	m.check(R.MinSpeed, "minSpeed", l)
	m.check(R.MaxSpeed, "maxSpeed", l)
	m.check(R.MinLimitedSpeed, "minLimitedSpeed", l)

	m.check(p.Crr, "rollingResistanceCoef", l)
	m.check(R.Cbf, "brakingFrictionCoef", l)
	m.check(R.Ccf, "turnFrictionCoef", l)
	m.check(p.CdA, "airDragCoef CdA", l)
	m.check(p.DrivetrainLoss, "drivetrainLoss", l)
	// m.check(R.HeavyBrakingPower, "heavyBrakingPower", l)
	m.check(p.UphillBreak.PowerLimit, "uphillBreak.powerLimit", l)
	m.check(p.UphillBreak.BreakDuration, "uphillBreak.breakDuration", l)
	m.check(p.UphillBreak.ClimbDuration, "uphillBreak.climbDuration", l)
	m.check(p.Filter.LevelFactor, "filter.levelFactor", l)
	m.check(p.Filter.LevelMax, "filter.levelMax", l)
	m.check(p.Filter.LevelMin, "filter.levelMin", l)
	m.check(p.Filter.MinSegDist, "minSegmentDist", l)
}

// UnitConversionIn --
func (p *Parameters) UnitConversionIn() {

	p.PowerIn = (100 - p.DrivetrainLoss) / 100
	p.PowerOut = 1 / p.PowerIn

	Q := &p.Powermodel
	Q.FlatSpeed *= kmh2ms
	Q.MaxPedaledSpeed *= kmh2ms
	Q.UphillPowerSpeed *= kmh2ms
	Q.UphillPowerGrade /= 100
	Q.DownhillPowerGrade /= 100
	//DrivetrainLoss is removed immediately and is returned to the results
	Q.FlatPower *= p.PowerIn
	Q.UphillPower *= p.PowerIn

	p.UphillBreak.PowerLimit /= 100
	p.UphillBreak.ClimbDuration *= min2sec
	p.UphillBreak.BreakDuration *= min2sec
	p.VerticalUpGrade /= 100
	p.Filter.InitialRelGrade /= 100
	p.Filter.MinRelGrade /= 100

	R := &p.Ride
	R.MaxSpeed *= kmh2ms
	R.MinSpeed *= kmh2ms
	R.MinLimitedSpeed *= kmh2ms
	R.PowerAcceMin /= 100
	R.PowerAcce /= 100
	R.PowerDece /= 100
	R.SteepDownhillGrade /= 100
	R.VelLimitGrade /= 100

}

// unitConversionOut --
func (p *Parameters) UnitConversionOut() {
	q := &p.Powermodel
	q.FlatSpeed *= ms2kmh
	q.UphillPowerSpeed *= ms2kmh
	q.MaxPedaledSpeed *= ms2kmh
	q.UphillPowerGrade *= 100
	q.DownhillPowerGrade *= 100
	q.FlatPower *= p.PowerOut
	q.UphillPower *= p.PowerOut

	p.UphillBreak.PowerLimit *= 100
	p.UphillBreak.ClimbDuration *= sec2min
	p.UphillBreak.BreakDuration *= sec2min
	p.VerticalUpGrade *= 100
	p.Filter.InitialRelGrade *= 100
	p.Filter.MinRelGrade *= 100
	p.MinPedaledGrade *= 100

	R := &p.Ride
	R.MaxSpeed *= ms2kmh
	R.MinLimitedSpeed *= ms2kmh
	R.MinSpeed *= ms2kmh
	R.PowerAcceMin *= 100
	R.PowerAcce *= 100
	R.PowerDece *= 100
	R.SteepDownhillGrade *= 100
	R.VelLimitGrade *= 100
}

// WriteJSON ---
func (p *Parameters) WriteJSON(l logger) error {
	if !p.ParamOutJSON {
		return nil
	}
	b, e := json.MarshalIndent(p, "", "\t")
	if e != nil {
		return l.Errorf("Param json file: %v", e)
	}
	jsonfile := p.ResultDir + p.RouteName + "_parameters" + ".json"
	if e := os.WriteFile(jsonfile, b, 0644); e != nil {
		return l.Errorf("Param json file: %v", e)
	}
	return nil
}
