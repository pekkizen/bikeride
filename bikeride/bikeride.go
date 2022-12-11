package main

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/pkg/profile"
	"os"

	"bikeride/gpx"
	"bikeride/logerr"
	"bikeride/motion"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
)

const ms2kmh = 3.6
const deBug = true

func main() {

	l := logerr.New()
	startTotal := l.TimeNow()

	if emptyCommandLine(os.Args, l) {
		return
	}
	par, e := param.New(os.Args, l)
	if e != nil {
		l.Err(e)
		return
	}
	if e := initLogger(par, l); e != nil {
		l.Err(e)
		return
	}
	if e := par.Check(l); e != nil {
		return
	}
	par.UnitConversionIn()

	gpxfile := par.GPXdir + par.GPXfile
	gpxs, e := gpx.New(gpxfile, par.GPXuseXMLparser, par.GPXignoreErrors)
	if e != nil {
		l.Err(e)
		return
	}
	start := l.TimeNow()
	cal := motion.Calculator()
	gen := power.RatioGenerator()

	rou, e := route.New(gpxs, par)
	if e != nil {
		l.Err(e)
		return
	}
	rou.SetupRoad(par)

	if e := setupEnvironment(cal, gen, par, rou, l); e != nil {
		l.Printf("%v", e)
		return
	}
	rou.Filter()

	if e := rou.SetupRide(cal, gen, par, l); e != nil {
		l.Printf("%v", errSystem(e, cal, l))
		return
	}
	rou.Ride(cal, par, l)
	rou.UphillBreaks(par)
	res := rou.Results(cal, par)

	if len(os.Args) > 2 && os.Args[2] == "-prof" {
		cpuProfile(res, rou, gpxs, cal, gen, par, l)
	}

	res.UnitConversionOut()
	par.UnitConversionOut()
	calctime := l.TimeSince(start)

	if e := cal.Error(); e != nil {
		l.Msg(0, e)
	}

	w, e := res.Writer(par)
	if e == nil {
		e = res.WriteTXT(par, w)
	}
	if e != nil {
		l.Err("results CSV:", e)
	}

	w, e = rou.Writer(par)
	if e == nil {
		e = rou.WriteCSV(par, w)
	}
	if e != nil {
		l.Err("route CSV:", e)
	}

	if par.Display {
		t := float64(l.TimeSince(startTotal)) / 1e6
		c := float64(calctime) / 1e6
		res.Display(par, l)
		l.Printf("Route & results calculation time:%6.1f %s\n", c, "ms")
		l.Printf("Total time with I/O:             %6.0f %s\n", t, "ms")
	}
	if e := par.WriteJSON(l); e != nil {
		l.Err(e)
	}
	resultJSON, e := writeResultJSON(res, par, l)
	if e != nil {
		l.Err(e)
	}
	if deBug {
		rou.Log(par, l)
		writeSHA(resultJSON, "Result JSON", l)
	}
}

func writeSHA(b []byte, s string, l *logerr.Logerr) {
	if b == nil {
		return
	}
	sha := l.Sprintf("%X", sha256.Sum256(b))
	l.Printf(s+" file SHA256:             %s ...\n", sha[0:4])
}

// devNull is for profiling/testing without actual write
type DevNull int

func (DevNull) Close() error {
	return nil
}
func (i DevNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func writeResultJSON(r *route.Results, p *param.Parameters, l *logerr.Logerr) ([]byte, error) {
	if !p.ResultJSON {
		return nil, nil
	}
	b, e := json.MarshalIndent(r, "", "\t")
	if e != nil {
		return nil, l.Errorf("Result JSON file: %v", e)
	}
	e = os.WriteFile(p.ResultDir+p.RouteName+"_results"+".json", b, 0644)
	if e != nil {
		return nil, l.Errorf("Result JSON file: %v", e)
	}
	return b, nil
}
func errSystem(err error, cal *motion.BikeCalc, l *logerr.Logerr) error {
	s := "System error: the ride could not be calculated from given parameters.\n"
	s += "Symptom/cause" + ": Help! can you free us from pedalling in this dark place!"
	s = l.Sprintf("%s%v", s, err)
	return l.Errorf("%s%v", s, cal.Error())
}

func emptyCommandLine(args []string, l *logerr.Logerr) bool {

	if len(args) == 1 {
		l.Printf(version + " - " + copyright + "\n" + licnote)
		s := "<ride parameter file>|-gpx <GPX file>|-cfg <config file>\n"
		l.Printf("\n\nUsage: %s "+s, args[0])
		return true
	}
	if args[1] == "-lic" || args[1] == "-license" {
		l.Printf("%s", lictext+disclaimer+golicense)
		return true
	}
	return false
}

func initLogger(p *param.Parameters, l *logerr.Logerr) error {

	l.SetLevel(p.LogLevel)
	l.SetMode(p.LogMode)
	if p.LogMode != 1 {
		return nil
	}
	if p.Logfile == "" {
		// l.SetMode(0)
		return nil
	}
	if err := l.SetOutput(p.ResultDir + p.Logfile); err != nil {
		return l.Errorf("Log file -%v", err)
	}
	l.Msg(1, "\nBikeride         "+l.TimeStamp())
	l.Msg(1, "Route file:     ", p.GPXfile)
	cfg := p.ConfigJSON
	if len(cfg) > 1 {
		cfg = "and " + cfg
	}
	l.Msg(1, "Parameter files:", p.RideJSON, cfg)
	return nil
}

func cpuProfile(res *route.Results, rou *route.Route, gpxs *gpx.GPX,
	cal *motion.BikeCalc, gen *power.Generator, par *param.Parameters,
	log *logerr.Logerr) {
	_ = res
	prof := profile.Start(profile.ProfilePath("."))
	millions := 10
	rounds := millions*1000*1000/rou.Segments() + 1
	log.Printf("Calculating %d million road segments. Wait... or press ctrl+c\n", millions)
	for i := 0; i < rounds; i++ {
		// gpxs,_ := gpx.New(par.GPXdir+par.GPXfile, par.GPXuseXMLparser, par.GPXignoreErrors)
		// _ = gpxs

		rou, _ = route.New(gpxs, par)
		rou.SetupRoad(par)
		rou.Filter()
		rou.SetupRide(cal, gen, par, log)
		rou.Ride(cal, par, log)
		// rou.UphillBreaks(par)
		res = rou.Results(cal, par)
		_ = res
		// res.WriteCSV(par, devNull(0))
		// writeResultJSON(res, par, log)
		// rou.WriteCSV(par, devNull(0))
	}
	prof.Stop()
}

