package main

import (
	"github.com/pkg/profile"

	"github.com/pekkizen/bikeride/gpx"
	"github.com/pekkizen/bikeride/logerr"
	"github.com/pekkizen/bikeride/param"
	"github.com/pekkizen/bikeride/power"
	"github.com/pekkizen/bikeride/route"

	"github.com/pekkizen/motion"
)

type devNull int

func (devNull) Close() error {
	return nil
}
func (devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}
func cpuProfile(gpz *gpx.GPX, cal *motion.BikeCalc,
	gen *power.Generator, par *param.Parameters, log *logerr.Logerr) *route.Results {
	writer := devNull(911)
	millions := 100
	var res *route.Results
	rou, _ := route.New(gpz, par)
	rounds := millions * 1000 * 1000 / rou.Segments()
	log.Printf("Calculating %d million road segments. Wait... or press ctrl+c\n", millions)
	rou0, _ := route.New(gpz, par)
	prof := profile.Start(profile.ProfilePath("."))
	for i := 0; i < rounds; i++ {
		// gpz, _ := gpx.New(p.GPXdir+p.GPXfile, p.GPXuseXMLparser, p.GPXignoreErrors)
		// gpx.ParseGPX(gpxbytes, gpz, true)
		// rou, _ = route.New(gpz, par)
		rou.NewCopy(rou0) // to avoid garbage collection timing
		rou.SetupRoad(par)
		rou.Filter()
		rou.SetupRide(cal, gen, par)
		rou.Ride(cal, par)
		// rou.UphillBreaks(p)
		res = rou.Results(cal, par, log)
		// _ = res
		res.WriteTXT(par, writer)
		// res.WriteJSON(writer)
		// rou.WriteCSV(par, writer)
	}
	prof.Stop()
	return res
}
