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

	// r := &p.Ride
	q := &p.Powermodel
	f := &p.Filter
	u := &p.UphillBreak

	p.GPXfile = ""
	p.GPXdir = "./gpx/"
	p.ResultDir = "./results/"
	p.RouteCSV = true
	p.ResultTXT = true
	p.ResultJSON = false
	p.ParamOutJSON = false
	p.Logfile = "log.txt"
	p.LogMode = -1
	p.LogLevel = -1
	p.UseCR = false
	p.CSVuseTab = true
	p.Display = true
	p.CheckParams = true
	p.ReportTech = false
	p.GPXuseXMLparser = false
	p.GPXignoreErrors = true

	// f.MinSegDist = 3
	f.DistFilterTol = -1
	f.DistFilterDist = -1
	// f.IpoRounds = -1
	// f.Backsteps = 2
	// f.IpoDist = 60
	// f.IpoSumDist = 180
	// f.InitialRelGrade = 7
	// f.MinRelGrade = 0.01
	f.LevelFactor = -1
	f.LevelMax = 4
	f.LevelMin = 1
	f.SmoothingWeight = -1
	f.SmoothingWeightDist = -1
	// f.MaxAcceptedGrade = -1

	q.PowermodelType = 1
	q.TailWindPower = 85
	q.HeadWindPower = 115
	q.SysTailwind = -5
	q.SysHeadwind = 5
	q.ExpDownhill = 2
	q.ExpUphill = 0.75
	q.ExpHeadwind = 1
	q.ExpTailwind = 1
	q.DownhillPower = -1
	q.DownhillTailwindPower = 5
	q.DownhillHeadwindPower = 100
	q.VerticalUpGrade = 8
	q.CUT = 1
	q.CUH = 1
	q.CDT = 1
	q.CDH = 1

	// r.MaxSpeed = 80
	// r.LimitTurnSpeeds = false
	// r.LimitDownSpeeds = true
	// r.LimitExitSpeeds = false
	// r.MinLimitedSpeed = 8
	// r.MinSpeed = -1
	// r.SteepDownhillGrade = -12
	// r.PowerAcce = 125
	// r.PowerAcceMin = 50
	// r.PowerDece = 85
	// r.VelDeceLim = 50
	// r.SpeedLimitGrade = -1

	u.PowerLimit = 90
	u.ClimbDuration = 0
	u.BreakDuration = 0

	p.AcceStepMode = -1
	p.DeltaVel = -1
	p.DeltaTime = -1
	p.VelSolver = -1
	p.VelTol = -1
	p.Bracket = -1
	p.VelErrors = false
	// p.UseVelTable = false

}

type attributes struct {
	min      float64
	max      float64
	unit     string
	notGiven float64
}

const mustGiven = 0.18471183528821574770795166386961

type attributesMap map[string]attributes

func (m attributesMap) put(key string, min, max float64, unit string, notGiven float64) {
	m[key] = attributes{min, max, unit, notGiven}
}

func (m attributesMap) check(val float64, key string, l logger) {
	a, found := m[key]
	if !found {
		l.Err("No attributes found for key" + key)
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
	l.Err("Out of range "+key+" =", val, "(range", a.min, "-", a.max, a.unit+
		" and not given =", a.notGiven, ")")
}

func setParamRanges() attributesMap {
	m := make(attributesMap, 100)

	m.put("weight.total", 40, 10000, "kg", mustGiven)
	// environment
	m.put("windCourse", -2, 360, "degrees", mustGiven)
	m.put("windSpeed", -10, 10, "m/s", mustGiven)
	m.put("airDensity", 0.72, 1.5, "", -1)
	m.put("gravity", 9.780, 9.833, "", -1)
	m.put("temperature", -25, 50, "deg C", mustGiven)
	m.put("airPressure", 950, 1085, "hPascals", -1)
	m.put("baseElevation", 0, 5000, "m", -1)

	// Powermodel
	m.put("flatGroundSpeed", 10, 60, "km/h", -1)
	m.put("flatGroundPower", 25, 600, "w", -1)
	m.put("verticalUpSpeed", 100, 2500, "m/h", -1)
	m.put("uphillPower", 75, 750, "w", -1)
	m.put("uphillPowerSpeed", 3, 30, "km/h", -1)
	m.put("uphillPowerGrade", 2, 10, "%", -1)

	m.put("downhillPower", 10, 25, "%", -1)
	m.put("maxPedaledSpeed", 15, 80, "km/h", mustGiven)
	m.put("verticalDownSpeed", 500, 6000, "m/h", -1)
	m.put("brakingDist", 5, 100, "m", -1)

	m.put("tailWindPower", 50, 120, "%", mustGiven)
	m.put("sysTailWind", -8, -2, "m/s", mustGiven)
	m.put("sysHeadwind", 2, 8, "m/s", mustGiven)
	m.put("headWindPower", 80, 150, "%", mustGiven)
	m.put("downhillHeadwindPower", 50, 125, "%", -1)
	m.put("downhillTailwindPower", 1, 20, "%", -1)

	// Bike
	m.put("rollingResistanceCoef", 0.0001, 0.04, "", -1)
	m.put("brakeRoadFriction", 0.05, 1, "", mustGiven)
	m.put("turnFrictionCoef", 0.05, 0.5, "", mustGiven)
	m.put("airDragCoef CdA", 0.001, 1.5, "", -1)
	m.put("drivetrainLoss", 0, 20, "%", mustGiven)
	m.put("weight.total", 40, 10000, "kg", -1)

	// Ride
	m.put("maxSpeed", 20, 1000, "km/h", mustGiven)
	m.put("minSpeed", 3, 20, "km/h", -1)
	m.put("minLimitedSpeed", 5, 15, "km/h", mustGiven)
	m.put("decelerationPower", 25, 95, "%", mustGiven)
	m.put("accelerationPower", 105, 150, "%", mustGiven)
	m.put("minAccelerationPower", 0, 200, "w", -1)
	m.put("keepEntrySpeed", 1, 25, "%", -1)
	m.put("velDeceLim", 0, 100, "", mustGiven)

	// uphill breaks
	m.put("uphillBreak.powerLimit", 75, 95, "%", -1)
	m.put("uphillBreak.breakDuration", 1, 20, "min", -1)
	m.put("uphillBreak.climbDuration", 5, 60, "min", -1)

	//Filter
	m.put("filter.minSegmentDist", 1, 100, "m", -1)

	m.put("filter.levelFactor", 0.1, 1, "", -1)
	m.put("filter.levelMax", 1, 30, "m", mustGiven)
	m.put("filter.levelMin", 0.25, 5, "m", mustGiven)

	m.put("filter.distInterpolateTol", 0.0, 50, "%", -1)
	m.put("filter.distInterpolateDist", 10, 2000, "m", mustGiven)

	m.put("filter.maxAcceptedGrade", 5, 45, "%", -1)

	m.put("filter.smoothingWeight", 0.05, 10000, "", -1)
	m.put("filter.smoothingWeightDist", 5, 200, "", -1)

	m.put("filter.interpolateRounds", 1, 100, "", -1)
	m.put("filter.interpolateSumDist", 50, 400, "m", mustGiven)
	m.put("filter.interpolateDist", 5, 200, "m", mustGiven)
	m.put("filter.initialRelativeGrade", .1, 15, "", mustGiven)
	m.put("filter.interpolateBacksteps", 0, 10, "", mustGiven)
	m.put("filter.minRelativeGrade", 0.01, 3, "", mustGiven)

	// calculation
	m.put("velSolver", 1, 7, "", -1)
	m.put("acceStepMode", 1, 3, "", -1)

	return m
}

func checkParamRanges(p *Parameters, m attributesMap, l logger) {

	e := &p.Environment
	m.check(e.WindCourse, "windCourse", l)
	m.check(e.WindSpeed, "windSpeed", l)
	m.check(e.AirDensity, "airDensity", l)
	m.check(e.Gravity, "gravity", l)
	m.check(e.Temperature, "temperature", l)
	m.check(e.AirPressure, "airPressure", l)
	m.check(e.BaseElevation, "baseElevation", l)

	m.check(p.Bike.Crr, "rollingResistanceCoef", l)
	m.check(p.Bike.CdA, "airDragCoef CdA", l)
	m.check(p.Bike.DrivetrainLoss, "drivetrainLoss", l)
	m.check(p.Bike.Cbf, "brakeRoadFriction", l)
	m.check(p.Bike.Ccf, "turnFrictionCoef", l)
	m.check(p.Bike.Weight.Total, "weight.total", l)

	q := &p.Powermodel
	m.check(q.FlatSpeed, "flatGroundSpeed", l)
	m.check(q.FlatPower, "flatGroundPower", l)
	m.check(q.VerticalUpSpeed, "verticalUpSpeed", l)
	m.check(q.UphillPower, "uphillPower", l)
	m.check(q.UphillPowerSpeed, "uphillPowerSpeed", l)
	m.check(q.UphillPowerGrade, "uphillPowerGrade", l)
	m.check(q.MaxPedaledSpeed, "maxPedaledSpeed", l)
	m.check(q.TailWindPower, "tailWindPower", l)
	m.check(q.SysTailwind, "sysTailWind", l)
	m.check(q.SysHeadwind, "sysHeadwind", l)
	m.check(q.HeadWindPower, "headWindPower", l)
	m.check(q.DownhillPower, "downhillPower", l)
	m.check(q.DownhillHeadwindPower, "downhillHeadwindPower", l)
	m.check(q.DownhillTailwindPower, "downhillTailwindPower", l)

	r := &p.Ride
	m.check(r.PowerDece, "decelerationPower", l)
	m.check(r.PowerAcce, "accelerationPower", l)
	m.check(r.PowerAcceMin, "minAccelerationPower", l)
	m.check(r.VelDeceLim, "velDeceLim", l)
	m.check(r.MinSpeed, "minSpeed", l)
	m.check(r.MaxSpeed, "maxSpeed", l)
	m.check(r.MinLimitedSpeed, "minLimitedSpeed", l)
	m.check(r.VerticalDownSpeed, "verticalDownSpeed", l)
	m.check(r.BrakingDist, "brakingDist", l)
	m.check(r.KeepEntrySpeed, "keepEntrySpeed", l)

	u := &p.UphillBreak
	m.check(u.PowerLimit, "uphillBreak.powerLimit", l)
	m.check(u.BreakDuration, "uphillBreak.breakDuration", l)
	m.check(u.ClimbDuration, "uphillBreak.climbDuration", l)

	f := &p.Filter
	m.check(f.LevelFactor, "filter.levelFactor", l)
	if f.LevelFactor > 0 {
		m.check(f.LevelMax, "filter.levelMax", l)
		m.check(f.LevelMin, "filter.levelMin", l)
	}
	m.check(f.MinSegDist, "filter.minSegmentDist", l)
	m.check(f.DistFilterTol, "filter.distInterpolateTol", l)
	if f.DistFilterTol >= 0 {
		m.check(f.DistFilterDist, "filter.distInterpolateDist", l)
	}
	m.check(f.MaxAcceptedGrade, "filter.maxAcceptedGrade", l)
	m.check(f.SmoothingWeight, "filter.smoothingWeight", l)
	if f.SmoothingWeight > 0 {
		m.check(f.SmoothingWeightDist, "filter.smoothingWeightDist", l)
	}
	m.check(float64(f.IpoRounds), "filter.interpolateRounds", l)
	if f.IpoRounds > 0 {
		m.check(f.InitialRelGrade, "filter.initialRelativeGrade", l)
		m.check(float64(f.Backsteps), "filter.interpolateBacksteps", l)
		m.check(f.MinRelGrade, "filter.minRelativeGrade", l)
		m.check(f.IpoSumDist, "filter.interpolateSumDist", l)
		m.check(f.IpoDist, "filter.interpolateDist", l)
	}
	// calculation
	m.check(float64(p.VelSolver), "velSolver", l)
	m.check(float64(p.AcceStepMode), "acceStepMode", l)
	// if p.AcceStepMode > 1 {
	// 	p.DeltaVel = p.DeltaTime * 0.5

	// }
}

func (p *Parameters) Check(l logger) error {
	l.SetPrefix("param:      ")
	q := &p.Powermodel
	u := &p.UphillBreak
	b := &p.Bike

	if q.FlatSpeed <= 0 && q.FlatPower <= 0 {
		l.Err("flatGroundSpeed <= 0 and flatGroundPower <= 0")
	}
	if q.VerticalUpSpeed <= 0 && q.UphillPower <= 0 {
		l.Err("verticalUpSpeed <= 0 and uphillPower <= 0")
	}
	if q.UphillPowerSpeed <= 0 && q.UphillPowerGrade <= 0 {
		l.Err("uphillPowerSpeed <= 0 and uphillPowerGrade <= 0")
	}
	// if Q.MaxPedaledSpeed <= 0 && Q.DownhillPowerGrade <= 0 {
	// 	l.Err("maxPedalledSpeed <= 0 and downhillPowerGrade <= 0")
	// }
	if b.CdA <= 0 && (q.FlatSpeed <= 0 || q.FlatPower <= 0) {
		l.Err("airDragCoef CdA <= 0 and (flatSpeed <= 0 or flatPower <= 0)")
	}
	if u.ClimbDuration > 0 && u.BreakDuration > 0 && u.BreakDuration > u.ClimbDuration {
		l.Err("uphillBreaks.breakDuration  > uphillBreaks.climbDuration")
	}
	if b.Weight.Total <= 0 {
		b.Weight.Total = b.Weight.Bike + b.Weight.Rider + b.Weight.Luggage
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

	p.PowerIn = (100 - p.Bike.DrivetrainLoss) / 100
	p.PowerOut = 1 / p.PowerIn

	r := &p.Ride
	q := &p.Powermodel
	f := &p.Filter
	u := &p.UphillBreak

	q.VerticalUpGrade /= 100
	q.FlatSpeed *= kmh2ms
	q.MaxPedaledSpeed *= kmh2ms
	q.UphillPowerSpeed *= kmh2ms
	q.UphillPowerGrade /= 100
	q.DownhillPowerGrade /= 100
	//DrivetrainLoss is removed immediately and returned to the results
	q.FlatPower *= p.PowerIn
	q.UphillPower *= p.PowerIn

	u.PowerLimit /= 100
	u.ClimbDuration *= min2sec
	u.BreakDuration *= min2sec

	f.InitialRelGrade /= 100
	f.MinRelGrade /= 100
	f.MaxAcceptedGrade /= 100
	f.DistFilterTol /= 100

	r.MaxSpeed *= kmh2ms
	r.MinSpeed *= kmh2ms
	r.MinLimitedSpeed *= kmh2ms
	r.PowerAcce /= 100
	r.PowerDece /= 100
	r.VelDeceLim /= 100
	r.KeepEntrySpeed /= 100
	r.SteepDownhillGrade /= 100
	r.SpeedLimitGrade /= 100
	r.PowerAcceMin *= p.PowerIn
}

func (p *Parameters) UnitConversionOut() {

	r := &p.Ride
	q := &p.Powermodel
	f := &p.Filter
	u := &p.UphillBreak

	q.VerticalUpGrade *= 100
	q.FlatSpeed *= ms2kmh
	q.UphillPowerSpeed *= ms2kmh
	q.MaxPedaledSpeed *= ms2kmh
	q.UphillPowerGrade *= 100
	q.DownhillPowerGrade *= 100
	q.MinPedaledGrade *= 100
	q.FlatPower *= p.PowerOut
	q.UphillPower *= p.PowerOut

	u.PowerLimit *= 100
	u.ClimbDuration *= sec2min
	u.BreakDuration *= sec2min

	f.InitialRelGrade *= 100
	f.MinRelGrade *= 100
	f.MaxAcceptedGrade *= 100
	f.DistFilterTol *= 100

	r.MaxSpeed *= ms2kmh
	r.MinLimitedSpeed *= ms2kmh
	r.MinSpeed *= ms2kmh
	r.PowerAcce *= 100
	r.PowerDece *= 100
	r.VelDeceLim *= 100
	r.KeepEntrySpeed *= 100
	r.SteepDownhillGrade *= 100
	r.SpeedLimitGrade *= 100
	r.PowerAcceMin *= p.PowerOut
}
