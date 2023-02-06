package main

import (
	"os"

	"bikeride/gpx"
	"bikeride/logerr"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
	"motion"
)

const (
	version   = "Bikeride v1.00"
	copyright = "Copyright 2022 Pekkizen. All rights reserved."
	licnote   = "Use of " + version + " is governed by GNU General Public License v3.0."
)
const ms2kmh = 3.6
const deBUG = true
const proFILE = true

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
	gpxfile := par.GPXdir + par.GPXfile
	gpxTrack, e := gpx.New(gpxfile, par.GPXuseXMLparser, par.GPXignoreErrors)
	if e != nil {
		l.Err(e)
		return
	}
	start := l.TimeNow()
	cal := motion.Calculator()
	gen := power.RatioGenerator()

	par.UnitConversionIn()
	rou, e := route.New(gpxTrack, par)
	if e != nil {
		l.Err(e)
		return
	}
	if e := setupEnvironment(cal, gen, par, rou, l); e != nil {
		l.Err(e)
		return
	}
	rou.SetupRoad(par)
	rou.Filter()

	if e := rou.SetupRide(cal, gen, par); e != nil {
		l.Err(errSystem(e, cal, l))
		return
	}
	rou.Ride(cal, par)
	rou.UphillBreaks(par)
	if par.LogMode >= 0 {
		rou.Log(par, l)
	}
	res := rou.Results(cal, par)

	if e := cal.Error(); e != nil {
		l.Err(e)
		return
	}
	calctime := l.TimeSince(start)

	if proFILE && len(os.Args) > 2 && os.Args[2] == "-prof" {
		cpuProfile(rou, gpxTrack, cal, gen, par, l, devNull(0))
	}
	par.UnitConversionOut()
	writeAllResults(par, l, res, rou)

	if par.Display {
		totTime := float64(l.TimeSince(startTotal)) / 1e6
		calctime := float64(calctime) / 1e6
		l.Printf("\nRoute & results calculation time:%6.1f %s\n", calctime, "ms")
		l.Printf("Total time with I/O:             %6.0f %s\n", totTime, "ms")
	}
}

func writeAllResults(par *param.Parameters, l *logerr.Logerr, res *route.Results, rou *route.Route) {
	if par.Display {
		res.Display(par, l)
		if deBUG {
			l.Printf("Result check number:        %d \n", res.CheckNumber())
		}
	}
	if par.ResultTXT {
		w, e := res.ResultWriter(par, "txt")
		if e == nil {
			e = res.WriteTXT(par, w)
		}
		if e != nil {
			l.Err("results TXT:", e)
		}
	}
	if par.RouteCSV {
		w, e := rou.RouteWriter(par)
		if e == nil {
			e = rou.WriteCSV(par, w)
		}
		if e != nil {
			l.Err("Route CSV:", e)
		}
	}
	if par.ResultJSON {
		w, e := res.ResultWriter(par, "json")
		if e == nil {
			e = res.WriteJSON(w)
		}
		if e != nil {
			l.Err("Results JSON:", e)
		}
	}
	if e := par.WriteJSON(l); e != nil {
		l.Err("Parameters JSON:", e)
	}
}

func errSystem(err error, cal *motion.BikeCalc, l *logerr.Logerr) error {
	s := "System error: the ride could not be calculated from the given parameters.\n"
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
	l.Msg(1, "\n"+version+"   "+l.TimeStamp())
	l.Msg(1, "Route file:     ", p.GPXfile)
	cfg := p.ConfigJSON
	if len(cfg) > 1 {
		cfg = "and " + cfg
	}
	l.Msg(1, "Parameter files:", p.RideJSON, cfg)
	return nil
}
