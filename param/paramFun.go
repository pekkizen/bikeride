package param

import (
	"encoding/json"
	"io"
	"os"
)

func New(args []string, l logger) (*Parameters, error) {

	p := &Parameters{}
	p.setSystemDefaultValues()

	cfgfile := getCommandLineArg("-cfg", args)
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

	rideJSON := args[1]
	bytes, err = os.ReadFile(rideJSON)
	if err != nil {
		return p, l.Errorf("Ride parameter file: %v", err)
	}
	if err = json.Unmarshal(bytes, &p); err != nil {
		return p, l.Errorf("File "+rideJSON+" - %v", err)
	}
	p.RideJSON = rideJSON

	gpxfile := getCommandLineArg("-gpx", args)
	if gpxfile != "" {
		if p.RouteName == "" {
			p.RouteName = routeNameFromFileName(gpxfile)
		}
		p.GPXfile = gpxfile
	}
	if p.GPXfile == "" {
		return p, l.Errorf("No GPX file given")
	}
	return p, nil
}

func getCommandLineArg(arg string, args []string) string {
	for i := 1; i < len(args)-1; i++ {
		if args[i] == arg {
			return args[i+1]
		}
	}
	return ""
}

func routeNameFromFileName(filename string) string {
	//take bytes before possible dot
	b := []byte(filename)
	i := 0
	for i < len(b) && b[i] != '.' {
		i++
	}
	return string(b[0:i])
}

func (p *Parameters) WriteJSON(writer io.WriteCloser) error {
	json, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return err
	}
	_, err = writer.Write(json)
	if err == nil {
		err = writer.Close()
	}
	return err
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
	p.LogMode = -1
	p.LogLevel = 1
	p.UseCR = true
	p.CSVuseTab = false
	p.Display = true
	p.CheckParams = true
	p.ReportTech = false

	p.Filter.IpoBackStepRounds = 2
	p.Filter.IpoDist = 0
	p.Filter.IpoSumDist = 0
	p.Filter.InitialRelGrade = 0
	p.Filter.MinRelGrade = 0
	p.Filter.Backsteps = 5
	p.Filter.LevelFactor = 0.5
	p.Filter.LevelMax = 4
	p.Filter.LevelMin = 1

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
	R.MinLimitedSpeed = 8
	R.MinSpeed = -1
	R.SteepDownhillGrade = -12
	R.PowerAcce = 125
	R.PowerAcceMin = 50
	R.PowerDece = 75
	R.VelDeceLim = 0.5
	R.SpeedLimitGrade = -1

	p.DeltaVel = 0.5
	p.DeltaTime = 1
	p.DiffCalc = 1
	p.VelSolver = 1
	p.NRtol = 0.05
	p.Bracket = 0.5
	p.VelErrors = false
	p.UseVelTable = false
	p.VerticalUpGrade = 7
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

func setParamRanges() attributesMap {
	m := make(attributesMap, 100)
	m.put("weight.total", 40, 10000, "kg", mustGiven)
	m.put("windCourse", 0, 360, "degrees", -1)
	m.put("windSpeed", -10, 10, "m/s", mustGiven)
	m.put("airDensity", 0.72, 1.5, "", -1)
	m.put("gravity", 9.780, 9.833, "", -1)
	m.put("temperature", -25, 45, "deg C", mustGiven)
	m.put("airPressure", 950, 1085, "hPascals", -1)
	m.put("baseElevation", 0, 5000, "m", mustGiven)

	m.put("flatGroundSpeed", 10, 60, "km/h", -1)
	m.put("flatGroundPower", 25, 600, "w", -1)
	m.put("verticalUpSpeed", 100, 2500, "m/h", -1)
	m.put("uphillPower", 75, 750, "w", -1)
	m.put("uphillPowerSpeed", 3, 25, "km/h", -1)
	m.put("uphillPowerGrade", 2, 10, "%", -1)

	m.put("downhillPower", 10, 25, "%", mustGiven)
	m.put("downhillPowerGrade", -8, -1.5, "%", -1)
	m.put("maxPedalledSpeed", 15, 80, "km/h", mustGiven)
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
	m.put("minAccelerationPower", 0, 200, "w", mustGiven)
	m.put("deceFreeWheelCutoff", 0, 1, "", mustGiven)

	m.put("rollingResistanceCoef", 0.001, 0.02, "", mustGiven)
	m.put("brakeRoadFriction", 0.1, 1, "", mustGiven)
	m.put("turnFrictionCoef", 0.05, 0.5, "", mustGiven)
	m.put("airDragCoef CdA", 0.01, 1.5, "", -1)
	m.put("drivetrainLoss", 0, 20, "%", mustGiven)

	m.put("maxSpeed", 20, 1000, "km/h", mustGiven)
	m.put("minSpeed", 2, 20, "km/h", -1)
	m.put("minLimitedSpeed", 5, 15, "km/h", mustGiven)
	m.put("uphillBreak.powerLimit", 75, 95, "%", 0)
	m.put("uphillBreak.breakDuration", 1, 20, "min", 0)
	m.put("uphillBreak.climbDuration", 5, 60, "min", 0)
	m.put("minSegmentDist", 0.5, 100, "m", mustGiven)
	m.put("filter.levelFactor", 0.1, 1, "", 0)
	m.put("filter.levelMax", 1, 30, "m", mustGiven)
	m.put("filter.levelMin", 0.25, 5, "m", mustGiven)
	m.put("filter.distInterpolationGrade", 1, 5, "%", 0)
	m.put("filter.interpolateSumDist", 50, 300, "m", 0)
	m.put("filter.maxAcceptedGrade", 5, 30, "%", 0)
	m.put("filter.smoothingWeight", 0.1, 20, "", 0)
	m.put("filter.interpolateBackstepRounds", 1, 100, "", 0)
	m.put("filter.initialRelativeGrade", 2, 15, "", 0)
	m.put("filter.interpolateBacksteps", 1, 10, "", 0)
	m.put("filter.minRelativeGrade", 0.01, 3, "", 0)
	
	return m
}

func checkParamRanges(p *Parameters, m attributesMap, l logger) {

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
	m.check(R.VerticalDownSpeed, "verticalDownSpeed", l)
	m.check(R.BrakingDist, "brakingDist", l)

	m.check(p.Crr, "rollingResistanceCoef", l)
	m.check(R.Cbf, "brakeRoadFriction", l)
	m.check(R.Ccf, "turnFrictionCoef", l)
	m.check(p.CdA, "airDragCoef CdA", l)
	m.check(p.DrivetrainLoss, "drivetrainLoss", l)

	m.check(p.UphillBreak.PowerLimit, "uphillBreak.powerLimit", l)
	m.check(p.UphillBreak.BreakDuration, "uphillBreak.breakDuration", l)
	m.check(p.UphillBreak.ClimbDuration, "uphillBreak.climbDuration", l)

	m.check(p.Filter.LevelFactor, "filter.levelFactor", l)
	m.check(p.Filter.LevelMax, "filter.levelMax", l)
	m.check(p.Filter.LevelMin, "filter.levelMin", l)
	m.check(p.Filter.MinSegDist, "minSegmentDist", l)
	m.check(p.Filter.DistFilterGrade, "filter.distInterpolationGrade", l)
	m.check(p.Filter.IpoSumDist, "filter.interpolateSumDist", l)
	m.check(p.Filter.MaxAcceptedGrade, "filter.maxAcceptedGrade", l)
	m.check(p.Filter.WeightedAvgWeight, "filter.smoothingWeight", l)
	m.check(float64(p.Filter.IpoBackStepRounds), "filter.interpolateBackstepRounds", l)
	m.check(p.Filter.InitialRelGrade, "filter.initialRelativeGrade", l)
	m.check(float64(p.Filter.Backsteps ), "filter.interpolateBacksteps", l)
	m.check(p.Filter.MinRelGrade , "filter.minRelativeGrade", l)
}


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
		m := setParamRanges()
		checkParamRanges(p, m, l)
	}
	if l.Errors() > 0 {
		return l.Errorf(" ")
	}
	return nil
}

func (p *Parameters) UnitConversionIn() {

	p.PowerIn = (100 - p.DrivetrainLoss) / 100 // **************
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
	p.Ride.PowerAcceMin *= p.PowerIn

	p.UphillBreak.PowerLimit /= 100
	p.UphillBreak.ClimbDuration *= min2sec
	p.UphillBreak.BreakDuration *= min2sec
	p.VerticalUpGrade /= 100
	p.Filter.InitialRelGrade /= 100
	p.Filter.MinRelGrade /= 100
	p.Filter.MaxAcceptedGrade /= 100
	p.Filter.DistFilterGrade /= 100
	R := &p.Ride
	R.MaxSpeed *= kmh2ms
	R.MinSpeed *= kmh2ms
	R.MinLimitedSpeed *= kmh2ms
	R.PowerAcce /= 100
	R.PowerDece /= 100
	R.SteepDownhillGrade /= 100
	R.SpeedLimitGrade /= 100
}

func (p *Parameters) UnitConversionOut() {
	Q := &p.Powermodel
	Q.FlatSpeed *= ms2kmh
	Q.UphillPowerSpeed *= ms2kmh
	Q.MaxPedaledSpeed *= ms2kmh
	Q.UphillPowerGrade *= 100
	Q.DownhillPowerGrade *= 100
	Q.FlatPower *= p.PowerOut
	Q.UphillPower *= p.PowerOut
	p.Ride.PowerAcceMin *= p.PowerOut

	p.UphillBreak.PowerLimit *= 100
	p.UphillBreak.ClimbDuration *= sec2min
	p.UphillBreak.BreakDuration *= sec2min
	p.VerticalUpGrade *= 100
	p.Filter.InitialRelGrade *= 100
	p.Filter.MinRelGrade *= 100
	p.Filter.MaxAcceptedGrade *= 100
	p.Filter.DistFilterGrade *= 100
	p.MinPedaledGrade *= 100

	R := &p.Ride
	R.MaxSpeed *= ms2kmh
	R.MinLimitedSpeed *= ms2kmh
	R.MinSpeed *= ms2kmh
	R.PowerAcce *= 100
	R.PowerDece *= 100
	R.SteepDownhillGrade *= 100
	R.SpeedLimitGrade *= 100
}

