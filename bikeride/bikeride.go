package main

// bike route simulator
import (
	"crypto/sha256"
	"io"
	"os"

	// "github.com/pkg/profile"

	"bikeride/gpx"
	"bikeride/logerr"
	"bikeride/motion"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
)

const (
	version   = "Bikeride v1.00"
	copyright = "Copyright 2022 Pekkizen. All rights reserved."
	licnote   = "Use of " + version + " is governed by GNU General Public License v3.0."
)
const ms2kmh = 3.6
const deBUG = true

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
	if e = initLogger(par, l); e != nil {
		l.Err(e)
		return
	}
	if e = par.Check(l); e != nil {
		return
	}
	gpxfile := par.GPXdir + par.GPXfile
	gpxs, e := gpx.New(gpxfile, par.GPXuseXMLparser, par.GPXignoreErrors)
	if e != nil {
		l.Err(e)
		return
	}
	start := l.TimeNow()
	cal := motion.Calculator()
	gen := power.RatioGenerator()

	par.UnitConversionIn()
	rou, e := route.New(gpxs, par)
	if e != nil {
		l.Err(e)
		return
	}
	if e = setupEnvironment(cal, gen, par, rou, l); e != nil {
		l.Err(e)
		return
	}
	rou.SetupRoad(par)
	rou.Filter()

	if e = rou.SetupRide(cal, gen, par); e != nil {
		l.Err(errSystem(e, cal, l))
		return
	}
	rou.Ride(cal, par)
	rou.UphillBreaks(par)

	if par.LogMode >= 0 {
		rou.Log(par, l)
	}
	res := rou.Results(cal, par)
	

	if e = cal.Error(); e != nil {
		l.Msg(0, e)
	}
	calctime := l.TimeSince(start)

	if len(os.Args) > 2 && os.Args[2] == "-prof" {
		cpuProfile(rou, gpxs, cal, gen, par, l)
	}
	par.UnitConversionOut()

	w, e := resultWriter(par, "txt")
	if e == nil {
		e = res.WriteTXT(par, w)
	}
	if e != nil {
		l.Err("results TXT:", e)
	}
	if w, e = routeWriter(par); e == nil {
		e = rou.WriteCSV(par, w)
	}
	if e != nil {
		l.Err("Route CSV:", e)
	}
	if e = par.WriteJSON(l); e != nil {
		l.Err("Parameters JSON:", e)
	}
	if par.Display {
		res.Display(par, l)
		t := float64(l.TimeSince(startTotal)) / 1e6
		c := float64(calctime) / 1e6
		l.Printf("Route & results calculation time:%6.1f %s\n", c, "ms")
		l.Printf("Total time with I/O:             %6.0f %s\n", t, "ms")
	}

	if !par.ResultJSON {
		return
	}
	if w, e = resultWriter(par, "json"); e != nil {
		l.Err("Results JSON:", e)
		return
	}
	if !deBUG {
		if _, e = res.WriteJSON(par, w); e != nil {
			l.Err("Results JSON:", e)
		}
		return
	}
	result, e := res.WriteJSON(par, w)
	if e != nil {
		l.Err("Results JSON:", e)
		return
	}
	sha := l.Sprintf("%X", sha256.Sum256(result))
	l.Printf("Result JSON file SHA256:    %s ...\n", sha[0:4])
}

func routeWriter(p *param.Parameters) (io.WriteCloser, error) {
	file := p.ResultDir + p.RouteName + ".csv"
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func resultWriter(p *param.Parameters, s string) (io.WriteCloser, error) {
	file := p.ResultDir + p.RouteName + "_results." + s
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func errSystem(err error, cal *motion.BikeCalc, l *logerr.Logerr) error {
	s := "System error: the ride could not be calculated from given parameters.\n"
	s += "Symptom/cause" + ": Help! can you free us from pedalling in this dark place!"
	s = l.Sprintf("%s%v", s, err)
	return l.Errorf("%s%v", s, cal.Error())
}

func emptyCommandLine(args []string, l *logerr.Logerr) bool {

	if len(args) == 1 {
		l.Printf("\n" + version + " - " + copyright + "\n" + licnote)
		l.Printf("\nYou can read it eg. from https://github.com/pekkizen/bikeride\n")
		s := " <ride parameter file>|-gpx <GPX route file>|-cfg <config file>\n"
		l.Printf("\nUsage: " + args[0] + s)
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
		return nil
	}
	if err := l.SetOutput(p.ResultDir + p.Logfile); err != nil {
		return l.Errorf("Log file -%v", err)
	}
	l.Msg(1, "\n"+version +"   "+l.TimeStamp())
	l.Msg(1, "Route file:     ", p.GPXfile)
	cfg := p.ConfigJSON
	if len(cfg) > 1 {
		cfg = "and " + cfg
	}
	l.Msg(1, "Parameter files:", p.RideJSON, cfg)
	return nil
}
