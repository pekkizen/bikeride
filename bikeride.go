package main

import (
	"io"
	"os"

	"github.com/pekkizen/bikeride/gpx"
	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/bikeride/param"
	"github.com/pekkizen/bikeride/power"
	"github.com/pekkizen/bikeride/route"

	"github.com/pekkizen/motion"
)

const (
	version   = "github.com/pekkizen/bikeride v1.00"
	copyright = "Copyright 2023 Pekkizen. All rights reserved."
	licnote   = "Use of " + version + " is governed by GNU General Public License v3.0."
)
const ms2kmh = 3.6
const development = true

func main() {
	l := logerr.New()
	if emptyCommandLine(os.Args, l) {
		return
	}
	p, e := param.New(os.Args, l)
	if e != nil {
		l.Err(e)
		return
	}
	if e := checkResultDir(p, l); e != nil {
		l.Err(e)
		return
	}
	if e := initLogger(p, l); e != nil {
		l.Err(e)
		return
	}
	if e := p.Check(l); e != nil {
		return
	}
	gpxfile := p.GPXdir + p.GPXfile
	gpz, e := gpx.New(gpxfile, p.GPXuseXMLparser, p.GPXignoreErrors)
	if e != nil {
		l.Err(e)
		return
	}
	p.UnitConversionIn()

	rou, e := route.New(gpz, p)
	if e != nil {
		l.Err(e)
		return
	}
	// gpz.TrkpSliceRelease()
	cal := motion.Calculator()
	gen := power.RatioGenerator()

	if e := setupSystem(cal, gen, p, rou, l); e != nil {
		l.Err(e)
		return
	}
	rou.SetupRoad(p)
	rou.Filter()

	if e := rou.SetupRide(cal, gen, p); e != nil {
		l.Err(sysErrorMsg(e, cal, l))
		return
	}
	rou.Ride(cal, p)
	rou.UphillBreaks(p)
	res := rou.Results(cal, p, l)
	_ = res
	if p.LogMode >= 0 {
		rou.Log(p, l)
	}
	if development && len(os.Args) > 2 && os.Args[2] == "-prof" {
		res = cpuProfile(gpz, cal, gen, p, l)
	}
	p.UnitConversionOut()
	writeAllResults(p, l, res, rou)
}

func writeAllResults(p *param.Parameters, l *logerr.Logerr, res *route.Results, rou *route.Route) {
	if p.Display {
		res.Display(p, l)
		if development {
			l.Printf("Result checksum:           %d \n", res.CheckSum()>>45)
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
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		return nil, e
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
	if e := os.MkdirAll(p.ResultDir, os.ModeDir); e != nil {
		// TODO this makes unusable dir in Linux, permissions
		return l.Errorf("Results dir: %v", e)
	}
	return nil
}

func sysErrorMsg(err error, cal *motion.BikeCalc, l *logerr.Logerr) string {
	s := "The ride could not be calculated from the given parameters. "
	s += "Symptom/cause: Help! can you free us from pedalling in this dark place!"
	s = l.Sprintf("%s%v", s, err)
	if development {
		s += l.Sprintf("\n%#v\n", *cal) // calculator dump
	}
	return s
}

func emptyCommandLine(args []string, l *logerr.Logerr) bool {
	if len(args) == 1 {
		l.Printf("\n" + version + " - " + copyright + "\n" + licnote)
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
