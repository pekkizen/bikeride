package main

import (
	"github.com/pkg/profile"

	"bikeride/gpx"
	"bikeride/logerr"
	"bikeride/param"
	"bikeride/power"
	"bikeride/route"
	"motion"
)

type devNull int

func (devNull) Close() error {
	return nil
}
func (devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}
func cpuProfile(rou *route.Route, gpxs *gpx.GPX, cal *motion.BikeCalc,
	gen *power.Generator, par *param.Parameters, log *logerr.Logerr) {
	if development {
		_ = devNull(0)
		prof := profile.Start(profile.ProfilePath("."))
		millions := 30
		rounds := millions * 1000 * 1000 / rou.Segments()
		log.Printf("Calculating %d million road segments. Wait... or press ctrl+c\n", millions)
		for i := 0; i < rounds; i++ {
			// gpxs, _ := gpx.New(par.GPXdir+par.GPXfile, par.GPXuseXMLparser, par.GPXignoreErrors)
			// _ = gpxs
			rou, _ := route.New(gpxs, par)
			rou.SetupRoad(par)
			rou.Filter()
			rou.SetupRide(cal, gen, par)
			rou.Ride(cal, par)
			rou.UphillBreaks(par)
			res := rou.Results(cal, par, log)
			_ = res
			res.WriteTXT(par, devNull(0))
			// res.WriteJSON(devNull(0))
			// rou.WriteCSV(par, devNull(0))
		}
		prof.Stop()
	}
}
