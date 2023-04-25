package main

import (
	"bikeride/gpx"
	"bikeride/logerr"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
	"io"
	"motion"
	"os"
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
	p, e := param.New(os.Args, l)
	if e != nil {
		l.Err(e)
		return
	}
	if e = checkResultDir(p, l); e != nil {
		l.Err(e)
		return
	}
	if e = initLogger(p, l); e != nil {
		l.Err(e)
		return
	}
	if e = p.Check(l); e != nil {
		return
	}
	gpxfile := p.GPXdir + p.GPXfile
	gpxTrack, e := gpx.New(gpxfile, p.GPXuseXMLparser, p.GPXignoreErrors)
	if e != nil {
		l.Err(e)
		return
	}
	start := l.TimeNow()
	p.UnitConversionIn()

	rou, e := route.New(gpxTrack, p)
	if e != nil {
		l.Err(e)
		return
	}
	cal := motion.Calculator()
	gen := power.RatioGenerator()

	if e = setupSystem(cal, gen, p, rou, l); e != nil {
		l.Err(e)
		return
	}
	rou.SetupRoad(p)
	rou.Filter()

	if e = rou.SetupRide(cal, gen, p); e != nil {
		l.Err(errSystem(e, cal, l))
		return
	}
	rou.Ride(cal, p)
	rou.UphillBreaks(p)
	if p.LogMode >= 0 {
		rou.Log(p, l)
	}
	res := rou.Results(cal, p, l)

	calctime := l.TimeSince(start)
	if deBUG && len(os.Args) > 2 && os.Args[2] == "-prof" {
		cpuProfile(rou, gpxTrack, cal, gen, p, l)
	}
	p.UnitConversionOut()
	writeAllResults(p, l, res, rou)

	if p.Display {
		totTime := float64(l.TimeSince(startTotal)) / 1e6
		calctime := float64(calctime) / 1e6
		l.Printf("\nRoute & results calculation time:%6.1f %s\n", calctime, "ms")
		l.Printf("Total time with I/O:             %6.0f %s\n", totTime, "ms")
	}
}

func writeAllResults(p *param.Parameters, l *logerr.Logerr, res *route.Results, rou *route.Route) {
	if p.Display {
		res.Display(p, l)
		if deBUG {
			l.Printf("Result checksum:            %d \n", res.CheckSum() >> 45)
		}
	}
	if p.ResultTXT {
		w, e := writer(p, "_results.txt")
		if e == nil {
			e = res.WriteTXT(p, w)
		}
		if e != nil {
			l.Err("Results TXT:", e)
		}
	}
	if p.ResultJSON {
		w, e := writer(p, "_results.json")
		if e == nil {
			e = res.WriteJSON(w)
		}
		if e != nil {
			l.Err("Results JSON:", e)
		}
	}
	if p.RouteCSV {
		w, e := writer(p, ".csv")
		if e == nil {
			e = rou.WriteCSV(p, w)
		}
		if e != nil {
			l.Err("Route CSV:", e)
		}
	}
	if p.ParamOutJSON {
		w, e := writer(p, "_parameters.json")
		if e == nil {
			e = p.WriteJSON(w)
		}
		if e != nil {
			l.Err("Parameters JSON:", e)
		}
	}
}

func writer(p *param.Parameters, s string) (io.WriteCloser, error) {
	name := p.ResultDir + p.RouteName + s
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func checkResultDir(p *param.Parameters, l *logerr.Logerr) error {
	if p.ResultDir == "" {
		return nil
	}
	if !p.ResultTXT && !p.RouteCSV && !p.ResultJSON && !p.ParamOutJSON {
		return nil
	}
	// if directory ResultDir exits, MkdirAll does nothing and returns nil
	if err := os.MkdirAll(p.ResultDir, os.ModeDir); err != nil {
		// TODO this makes unusable dir in Linux, permissions
		return l.Errorf("Results dir: %v", err)
	}
	return nil
}

func errSystem(err error, cal *motion.BikeCalc, l *logerr.Logerr) error {
	s := "The ride could not be calculated from the given parameters. "
	s += "Symptom/cause" + ": Help! can you free us from pedalling in this dark place!"
	s = l.Sprintf("%s%v", s, err)
	if e := cal.Error(); e != nil {
		s += e.Error()
	}
	return l.Errorf("%s", s)
}

func emptyCommandLine(args []string, l *logerr.Logerr) bool {
	if len(args) == 1 {
		l.Printf("\n" + version + " - " + copyright + "\n" + licnote)
		// l.Printf("\nYou can read it eg. from https://github.com/pekkizen/bikeride\n")
		s := " <ride parameter file>|-gpx <GPX route file>|-cfg <config file>\n"
		l.Printf("\n\nUsage: " + args[0] + s)
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
	l.Msg(1, "\n"+version+"   "+l.TimeStamp())
	l.Msg(1, "Route file:     ", p.GPXfile)
	cfg := p.ConfigJSON
	if len(cfg) > 1 {
		cfg = "and " + cfg
	}
	l.Msg(1, "Parameter files:", p.RideJSON, cfg)
	return nil
}
